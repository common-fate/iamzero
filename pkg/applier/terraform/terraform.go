package applier

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/common-fate/iamzero/pkg/applier"
	"github.com/common-fate/iamzero/pkg/policies"
	"github.com/common-fate/iamzero/pkg/recommendations"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// TerraformFinding is proposed Terraform source code changes recommended by IAM Zero
type TerraformFinding struct {
	FindingID       string                    `json:"findingId"`
	Role            string                    `json:"role"`
	Recommendations []TerraformRecommendation `json:"recommendations"`
}

type TerraformRecommendation struct {
	Type       string               `json:"type"`
	Statements []TerraformStatement `json:"statements"`
}

type TerraformStatement struct {
	Resources []TerraformResource `json:"resources"`
	Actions   []string            `json:"actions"`
}

type TerraformResource struct {
	Reference string  `json:"reference"`
	ARN       *string `json:"arn,omitempty"`
}

type StateFile struct {
	Values StateFileValues `json:"values"`
}

type StateFileValues struct {
	RootModule RootModule `json:"root_module"`
}
type RootModule struct {
	Resources    []StateFileResource `json:"resources"`
	ChildModules []ChildModuleModule `json:"child_modules"`
}

type ChildModuleModule struct {
	Resources []StateFileResource `json:"resources"`
	Address   string              `json:"address"`
}

type StateFileResource struct {
	Type    string             `json:"type"`
	Name    string             `json:"name"`
	Address string             `json:"address"`
	Values  StateFileAttribute `json:"values"`
}

type StateFileAttribute struct {
	Arn   string   `json:"arn"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

type StateFileResourceBlock struct {
	StateFileResource
	FilePath      string
	ParentAddress string
}

type FileHandler struct {
	HclFiles map[string]*hclwrite.File
}

type AwsIamBlock struct {
	*hclwrite.Block
}

type TerraformIAMPolicyApplier struct {
	applier.AWSIAMPolicyApplier
	Finding     *TerraformFinding
	FileHandler *FileHandler
}

var ROOT_TERRAFORM_FILE = "main.tf"

func (t TerraformIAMPolicyApplier) GetProjectName() string { return "Terraform" }
func (t TerraformIAMPolicyApplier) EvaluatePolicy(policy *recommendations.Policy, actions []recommendations.AWSAction) error {
	// Generate Finding from the context
	t.calculateTerraformFinding(policy, actions)
	return nil
}
func (t TerraformIAMPolicyApplier) Init() error {
	// Init File handler to manage reading and writing
	t.FileHandler = &FileHandler{HclFiles: make(map[string]*hclwrite.File)}
	return nil
}

func (t TerraformIAMPolicyApplier) Detect() bool {
	_, errTf := os.Stat(t.getRootFilePath())
	return os.IsExist(errTf)
}

func (t TerraformIAMPolicyApplier) Plan() (*applier.PendingChanges, error) {
	return t.FileHandler.PlanTerraformFinding(t.Finding)
}

func (t TerraformIAMPolicyApplier) Apply(changes *applier.PendingChanges) error {
	// Writes the changes to the files
	for _, change := range *changes {
		err := ioutil.WriteFile(change.Path, []byte(change.Contents), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t TerraformIAMPolicyApplier) getRootFilePath() string {
	return path.Join(t.ProjectPath, ROOT_TERRAFORM_FILE)
}
func (t TerraformIAMPolicyApplier) calculateTerraformFinding(policy *recommendations.Policy, actions []recommendations.AWSAction) {

	terraformFinding := TerraformFinding{
		FindingID: policy.ID,
		Role:      policy.Identity.Role,

		Recommendations: []TerraformRecommendation{
			// {
			// 	Type: "IAMInlinePolicy",
			// 	Statements: []TerraformStatement{
			// 		{
			// 			Resources: []TerraformResource{
			// 				{
			// 					Reference: bucketArn,
			// 					Type:      "AWS::S3::Bucket", ARN: &bucketArn,
			// 				},
			// 			},
			// 			Actions: actionsDemo,
			// 		},
			// 	},
			// },
		},
	}

	// I copied this and modified it from the CDK example, it is subject to the same TODO comments as CDK above
	for _, alert := range actions {
		if alert.Enabled && len(alert.Recommendations) > 0 {
			rec := TerraformRecommendation{
				Type:       "IAMInlinePolicy",
				Statements: []TerraformStatement{},
			}
			advisory := alert.GetSelectedAdvisory()
			for _, description := range advisory.Details().Description {
				policy, ok := description.Policy.(policies.AWSIAMPolicy)
				if ok {
					for _, s := range policy.Statement {
						terraformStatement := TerraformStatement{
							Actions: s.Action,
						}
						for _, resource := range alert.Resources {

							var terraformResource TerraformResource
							if resource.CDKResource == nil {

								terraformResource = TerraformResource{
									Reference: "IAM",
									ARN:       &resource.ARN,
								}
							}
							terraformStatement.Resources = append(terraformStatement.Resources, terraformResource)
						}
						rec.Statements = append(rec.Statements, terraformStatement)
					}
				}
			}
			terraformFinding.Recommendations = append(terraformFinding.Recommendations, rec)
		}
	}
	t.Finding = &terraformFinding
}

func (sfr StateFileResource) IsInRoot() bool {
	return strings.Split(sfr.Address, ".")[0] != "module"
}

func GenerateVariableName(resourceName string, propertyType string) string {
	return strings.Join([]string{"iamzero-variable", resourceName, propertyType}, "_")
}

func GenerateOutputName(resourceName string, propertyType string) string {
	return strings.Join([]string{"iamzero-output", resourceName, propertyType}, "_")
}

func TraversalFromAddress(address string) hcl.Traversal {
	splitAddress := strings.Split(address, ".")
	traversal := hcl.Traversal{
		hcl.TraverseRoot{Name: splitAddress[0]},
	}
	for _, val := range splitAddress[1:] {
		traversal = append(traversal, hcl.TraverseAttr{Name: val})
	}
	return traversal
}

func AddInputToModuleDeclaration(block *hclwrite.Block, variableName string, resourcePath string) {
	block.Body().SetAttributeTraversal(variableName, TraversalFromAddress(resourcePath))
}

func AppendVariableBlockIfNotExist(body *hclwrite.Body, name string, description string) string {
	block := FindBlockWithMatchingLabel(body.Blocks(), name)
	if block != nil {
		return name
	}
	AppendVariableBlock(body, name, description)
	return name

}
func AppendVariableBlock(body *hclwrite.Body, name string, description string) {
	body.AppendNewline()
	newBlock := hclwrite.NewBlock("variable", []string{name})
	newBlock.Body().SetAttributeValue("type", cty.StringVal("string"))
	newBlock.Body().SetAttributeValue("description", cty.StringVal(description))
	body.AppendBlock(newBlock)
	body.AppendNewline()
}

func AppendOutputBlockIfNotExist(body *hclwrite.Body, name string, description string, value string) string {
	block := FindBlockWithMatchingValueAttribute(body.Blocks(), value)
	if block != nil {
		return block.Labels()[0]
	}
	AppendOutputBlock(body, name, description, value)
	return name
}
func AppendOutputBlock(body *hclwrite.Body, name string, description string, value string) {
	body.AppendNewline()
	newBlock := hclwrite.NewBlock("output", []string{name})
	t := hclwrite.Token{Type: hclsyntax.TokenType('Q'), Bytes: []byte(value)}
	toks := hclwrite.Tokens{&t}
	newBlock.Body().SetAttributeRaw("value", toks)
	newBlock.Body().SetAttributeValue("description", cty.StringVal(description))
	body.AppendBlock(newBlock)
	body.AppendNewline()
}
func (fh FileHandler) OpenFile(path string, createIfNotExist bool) (*hclwrite.File, error) {
	// Open file will open or create an empty hclfile
	// nothing is written to the filesystem by this method
	if val, ok := fh.HclFiles[path]; ok {
		return val, nil
	}
	var err error
	_, fh.HclFiles[path], err = OpenAndParseHclFile(path, createIfNotExist)
	if err != nil {
		return nil, err
	}
	return fh.HclFiles[path], nil

}
func (fh FileHandler) FindAndRemovePolicyAttachmentsForRole(stateFile *StateFile, stateFileRole StateFileResource) {
	attachments := stateFile.FindPolicyAttachmentsInStateFileByRoleName(stateFileRole.Values.Name)
	for key, element := range attachments {
		// for each terraform file, update all the modules
		hclFile, _ := fh.OpenFile(key, false)
		for _, stateFileResource := range element {
			policyAttachmentBlock := FindBlockByModuleAddress(hclFile.Body().Blocks(), stateFileResource.Type+"."+stateFileResource.Name)
			RemovePolicyAttachmentRole(policyAttachmentBlock, stateFileRole)
		}
	}

}
func (fh FileHandler) PendingChanges() *applier.PendingChanges {
	/*
		return the formatted bytes for each open file
	*/
	pc := applier.PendingChanges{}
	for key, val := range fh.HclFiles {
		pc = append(pc, applier.PendingChange{Path: key, Contents: string(hclwrite.Format(val.Bytes()))})
	}
	return &pc
}
func (fh FileHandler) PlanTerraformFinding(finding *TerraformFinding) (*applier.PendingChanges, error) {
	stateFile, err := parseTerraformState()

	if err != nil {
		return &applier.PendingChanges{}, err
	}

	iamRoleStateFileResource, err := stateFile.FindResourceInStateFileByArn(finding.Role)
	if err != nil {
		return &applier.PendingChanges{}, err
	}
	hclFile, err := fh.OpenFile(iamRoleStateFileResource.FilePath, false)
	if err != nil {
		return &applier.PendingChanges{}, err
	}
	awsIamBlock := FindIamRoleBlockByModuleAddress(hclFile.Body().Blocks(), iamRoleStateFileResource.Type+"."+iamRoleStateFileResource.Name)

	if awsIamBlock != nil {
		// Remove any managed policy attachments for this role
		fh.FindAndRemovePolicyAttachmentsForRole(&stateFile, iamRoleStateFileResource.StateFileResource)

		// Apply the finding by appending inline policies to the role
		fh.ApplyFindingToBlock(awsIamBlock, iamRoleStateFileResource, hclFile, finding, &stateFile)
		return fh.PendingChanges(), nil
	}
	// If we don't find the matching role, either something went wrong in our code, or the statefile doesn't match the source code.
	// the user probably needs to run `terraform plan` again
	return &applier.PendingChanges{}, fmt.Errorf("an error occurred finding the matching iam role in your terraform project, your state file may be outdated, try running 'terraform plan'")

}
func (fh FileHandler) ApplyFindingToBlock(awsIamBlock *AwsIamBlock, iamRoleStateFileResource StateFileResourceBlock, hclFile *hclwrite.File, finding *TerraformFinding, stateFile *StateFile) error {
	newInlinePolicies := []*hclwrite.Block{}
	existingInlinePoliciesToRemove := []*hclwrite.Block{}
	for _, nestedBlock := range awsIamBlock.Body().Blocks() {
		if IsBlockInlinePolicy(nestedBlock) {
			existingInlinePoliciesToRemove = append(existingInlinePoliciesToRemove, nestedBlock)
		}
	}

	for _, reccomendation := range finding.Recommendations {
		for i, statement := range reccomendation.Statements {
			newBlock := hclwrite.NewBlock("inline_policy", nil)
			actionsJson, _ := json.Marshal(statement.Actions)
			arn := ""

			// The ARN from the recommendation can contain "/*" on the end for an s3 bucket, to look this up in
			// @TODO probably need to verify whether we could get specific things here(as in specific objects in a bucket)

			splitArn := statement.Resources[0].SplitArn()
			awsResource, err := stateFile.FindResourceInStateFileByArn(splitArn[0])
			root, _ := fh.OpenFile(ROOT_TERRAFORM_FILE, false)
			if err == nil {
				/*
					THE BELOW SCENARIOS ONLY SUPPORT A FLAT PROJECT STRUCTURE WHERE THERE IS ONLY 1 LEVEL OF MODULE ABSTRACTION

					MAIN.TF
						->MODULES
							->EC2
								MAIN.TF

				*/
				if awsResource.Address == iamRoleStateFileResource.Address {
					// both in same file
					// standard method
					arn = awsResource.Address + ".arn"
					// if there is a specific resource then join it to the resource arn
					if len(splitArn) > 1 && splitArn[1] != "*" {
						// This adds a join statement to the terraform so that we refer to the correct bucket arn but add the specific resource correctly if it was specified in the finding
						// https://www.terraform.io/docs/language/functions/join.html
						arn = fmt.Sprintf(`join("/", [%s,"%s"])`, arn, strings.Join(splitArn[1:], "/"))
					}

				} else if !awsResource.IsInRoot() && !iamRoleStateFileResource.IsInRoot() {
					// resources are in different files
					// create and output for the resource
					// create a variable for the role module
					//do the plumbing
					outputsFilePath := filepath.Join(filepath.Dir(awsResource.FilePath), "outputs.tf")
					outputsFile, err := fh.OpenFile(outputsFilePath, true)
					if err != nil {
						fmt.Println(err)
					}

					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.Name, "arn"), "IAMZero generated output for resource", strings.Trim(awsResource.Address, awsResource.ParentAddress+".")+".arn")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(root.Body().Blocks(), awsResource.FilePath)
					resourcePathInRootModule := ""
					if moduleDefinitionInRoot != nil {

						resourcePathInRootModule = strings.Join([]string{"module", moduleDefinitionInRoot.Labels()[0], outputName}, ".")
					}

					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(iamRoleStateFileResource.FilePath), "variables.tf")
					variablesFile, err := fh.OpenFile(variableFilePath, true)
					if err != nil {
						fmt.Println(err)
					}

					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.Name, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot = FindModuleBlockBySourcePath(root.Body().Blocks(), iamRoleStateFileResource.FilePath)
					if moduleDefinitionInRoot != nil {
						AddInputToModuleDeclaration(moduleDefinitionInRoot, variableName, resourcePathInRootModule)
						arn = "var." + variableName
					}

				} else if !awsResource.IsInRoot() && iamRoleStateFileResource.IsInRoot() {

					// role is in root,  resource is in another file
					// create an output from the resource definition
					// refer to it in the root inline policy

					outputsFilePath := filepath.Join(filepath.Dir(awsResource.FilePath), "outputs.tf")
					outputsFile, err := fh.OpenFile(outputsFilePath, true)
					if err != nil {
						fmt.Println(err)
					}
					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.Name, "arn"), "IAMZero generated output for resource", strings.Trim(awsResource.Address, awsResource.ParentAddress+".")+".arn")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(root.Body().Blocks(), awsResource.FilePath)
					if moduleDefinitionInRoot != nil {

						arn = strings.Join([]string{"module", moduleDefinitionInRoot.Labels()[0], outputName}, ".")
					}

				} else if awsResource.IsInRoot() && !iamRoleStateFileResource.IsInRoot() {
					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(iamRoleStateFileResource.FilePath), "variables.tf")
					variablesFile, err := fh.OpenFile(variableFilePath, true)
					if err != nil {
						fmt.Println(err)
					}
					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.Name, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(root.Body().Blocks(), iamRoleStateFileResource.FilePath)
					if moduleDefinitionInRoot != nil {
						AddInputToModuleDeclaration(moduleDefinitionInRoot, variableName, awsResource.Address+".arn")
						arn = "var." + variableName
					}
				}

			} else {
				// add quotes around it to make it valid for our use, maybe a json stringify equivalent would be good here to add the quotes robustly
				arn = fmt.Sprintf(`"%s"`, *statement.Resources[0].ARN)
				fmt.Printf("Failed to find matching resource in state file for ARN(%s) in (%s) using ARN reference directly\n", arn, stateFile)
			}

			setInlinePolicyIamPolicy(newBlock, string(actionsJson), arn, "iamzero-generated-iam-policy-"+fmt.Sprint(i))
			newInlinePolicies = append(newInlinePolicies, newBlock)
		}
	}
	// some logic probably works out whether all existing policies need to go
	for _, blockToRemove := range existingInlinePoliciesToRemove {
		awsIamBlock.Body().RemoveBlock(blockToRemove)
	}
	// add the new blocks(inline policies) to this role
	for _, blockToAdd := range newInlinePolicies {
		awsIamBlock.Body().AppendBlock(blockToAdd)
	}
	return nil

}

func RemovePolicyAttachmentRole(block *hclwrite.Block, stateFileRole StateFileResource) {
	line := string(block.Body().GetAttribute("roles").Expr().BuildTokens(hclwrite.Tokens{}).Bytes())
	line = strings.Trim(line, " []")
	roles := strings.Split(line, ",")
	filteredRoles := []string{}
	for _, role := range roles {
		// will either be a string litteral rolename or a reference to a role in tf
		trimmedRole := strings.Trim(role, " ")
		if !(strings.Contains(trimmedRole, stateFileRole.Type+"."+stateFileRole.Name) || trimmedRole == `"`+stateFileRole.Values.Name+`"`) {
			filteredRoles = append(filteredRoles, trimmedRole)
		}
	}

	t := hclwrite.Token{Type: hclsyntax.TokenType('Q'), Bytes: []byte(fmt.Sprintf(`[%s]`, strings.Join(filteredRoles, ",")))}
	toks := hclwrite.Tokens{&t}
	block.Body().SetAttributeRaw("roles", toks)
}
func FindIamRoleBlockByModuleAddress(blocks []*hclwrite.Block, moduleAddress string) *AwsIamBlock {
	//filters the blocks to find the target block by matching the address
	// and address looks like "aws_iam_role.iamzero-overprivileged-role"

	// for multiple files, this address will refer to the file by is path e.g "modules.ec2.aws_iam_role.role-name"
	for _, block := range blocks {
		if IsBlockAwsIamRole(block) {
			if resourceLabelToString(block.Labels()) == moduleAddress {
				return &AwsIamBlock{block}
			}
		}
	}
	return nil
}

// compares the "source" attribute of a module block in terraform, this accepts a full filepath or the directory path to the module file
func FindModuleBlockBySourcePath(blocks []*hclwrite.Block, moduleFolderPath string) *hclwrite.Block {
	for _, block := range blocks {
		if block.Type() == "module" {
			source := block.Body().GetAttribute("source")
			if source != nil && filepath.Dir(strings.Trim(string(block.Body().GetAttribute("source").Expr().BuildTokens(nil).Bytes()), ` "`)) == filepath.Dir(moduleFolderPath) {
				return block
			}
		}
	}
	return nil
}

func FindBlockByModuleAddress(blocks []*hclwrite.Block, moduleAddress string) *hclwrite.Block {
	for _, block := range blocks {
		if resourceLabelToString(block.Labels()) == moduleAddress {
			return block
		}
	}
	return nil
}

func FindBlockWithMatchingValueAttribute(blocks []*hclwrite.Block, valueString string) *hclwrite.Block {
	for _, block := range blocks {
		value := block.Body().GetAttribute("value")
		if value != nil && strings.Trim(string(value.Expr().BuildTokens(nil).Bytes()), " ") == valueString {
			return block
		}
	}
	return nil
}
func FindBlockWithMatchingLabel(blocks []*hclwrite.Block, name string) *hclwrite.Block {
	for _, block := range blocks {
		if len(block.Labels()) > 0 {
			value := block.Labels()[0]
			if value == name {
				return block
			}
		}
	}
	return nil
}

func resourceLabelToString(labels []string) string {
	return strings.Join(labels[:], ".")
}

func (tr TerraformResource) SplitArn() []string {
	return strings.Split(*tr.ARN, "/")
}

func (s StateFile) FindResourceInStateFileByArn(arn string) (StateFileResourceBlock, error) {
	return s.FindResourceInStateFileBase(arn, func(s StateFileResource) string {
		return s.Values.Arn
	})
}
func (s StateFile) FindResourceInStateFileByName(name string) (StateFileResourceBlock, error) {
	return s.FindResourceInStateFileBase(name, func(s StateFileResource) string {
		return s.Values.Name
	})
}
func sliceContains(slice []string, compareTo string) bool {
	for _, elem := range slice {
		if elem == compareTo {
			return true
		}
	}
	return false
}
func (s StateFile) FindPolicyAttachmentsInStateFileByRoleName(name string) map[string][]StateFileResource {
	terraformFilePath := ROOT_TERRAFORM_FILE
	resourceType := "aws_iam_policy_attachment"
	matches := make(map[string][]StateFileResource)
	for _, r := range s.Values.RootModule.Resources {
		if r.Type == resourceType && sliceContains(r.Values.Roles, name) {
			if matches[terraformFilePath] == nil {
				matches[terraformFilePath] = []StateFileResource{}
			}
			matches[terraformFilePath] = append(matches[terraformFilePath], r)
		}
	}
	for _, module := range s.Values.RootModule.ChildModules {
		for _, r := range module.Resources {
			if r.Type == resourceType && sliceContains(r.Values.Roles, name) {
				terraformFilePath = filepath.Join("modules", filepath.Join(strings.Split(module.Address, ".")[1:]...), terraformFilePath)
				if matches[terraformFilePath] == nil {
					matches[terraformFilePath] = []StateFileResource{}
				}
				matches[terraformFilePath] = append(matches[terraformFilePath], r)
			}
		}

	}
	return matches
}

func (s StateFile) FindResourceInStateFileBase(value string, attributefn func(s StateFileResource) string) (StateFileResourceBlock, error) {
	/*
		I added this to allow matching of any of the properties of the resource by passing in a function to select the attribute
	*/
	terraformFilePath := ROOT_TERRAFORM_FILE
	for _, r := range s.Values.RootModule.Resources {
		if attributefn(r) == value {
			return StateFileResourceBlock{r, terraformFilePath, "."}, nil
		}
	}
	for _, module := range s.Values.RootModule.ChildModules {
		for _, r := range module.Resources {
			if attributefn(r) == value {

				terraformFilePath = filepath.Join("modules", filepath.Join(strings.Split(module.Address, ".")[1:]...), terraformFilePath)
				return StateFileResourceBlock{r, terraformFilePath, module.Address}, nil
			}
		}

	}
	return StateFileResourceBlock{StateFileResource{}, "", ""}, fmt.Errorf("not found")
}
func parseTerraformState() (StateFile, error) {
	// need to be in the root of the terraform repo for this to work
	// doing this gets the correct state for us,alternative would be to copy this code from teh terraform repo
	out, err := exec.Command("terraform", "show", "-json").Output()
	if err != nil {
		return StateFile{}, err
	}
	return MarshalStateFileToGo(out)
}

func MarshalStateFileToGo(stateFileBytes []byte) (StateFile, error) {
	var stateFile StateFile
	err := json.Unmarshal(stateFileBytes, &stateFile)
	if err != nil {
		return StateFile{}, err
	}
	return stateFile, nil
}

func StringCompareAttributeValue(attribute *hclwrite.Attribute, compareTo string) bool {
	// Strip any leading or training whitespace from the Expression value or the attribute
	// check for an exact match
	att := strings.Trim(string(attribute.Expr().BuildTokens(hclwrite.Tokens{}).Bytes()), " ")
	if strings.Trim(att, `"`) == att {
		// attribute is likely a function or reference not a string litteral
		return att == compareTo
	} else {
		// A valid name will include "" quotes so these are added around the input string being compared
		return strings.Trim(string(attribute.Expr().BuildTokens(hclwrite.Tokens{}).Bytes()), " ") == fmt.Sprintf(`"%s"`, compareTo)
	}

}

func setInlinePolicyIamPolicy(block *hclwrite.Block, action string, resource string, name string) {
	// @TODO if hclwrite add a simple way to write function values like this we may switch over,
	// However for now it seems this is the simplest way to add a function block to HCL using the hclwite package
	t := hclwrite.Token{Type: hclsyntax.TokenType('Q'), Bytes: []byte(fmt.Sprintf(`jsonencode({
        Version = "2012-10-17"
        Statement = [
          {
            Action   = %s
            Effect   = "Allow"
            Resource = %s
          },
        ]
      })`, action, resource))}
	toks := hclwrite.Tokens{&t}
	block.Body().SetAttributeRaw("policy", toks)
	block.Body().SetAttributeValue("name", cty.StringVal(name))
}

/*
if createIfNotExist is true then this function will return a new empty hclwrite file if it is not found at the path

returns fileCreated(bool), file, error
*/
func OpenAndParseHclFile(filePath string, createIfNotExist bool) (bool, *hclwrite.File, error) {

	// read file bytes
	src, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) && createIfNotExist {
			return true, hclwrite.NewEmptyFile(), nil
		}
		return false, hclwrite.NewEmptyFile(), fmt.Errorf("failed to read file %s because %s", filePath, err)
	}
	hclfile, diagnostics := hclwrite.ParseConfig(src, filePath, hcl.InitialPos)
	if diagnostics != nil && diagnostics.HasErrors() {
		hclErrors := diagnostics.Errs()
		return false, hclwrite.NewEmptyFile(), fmt.Errorf("failed to parse hcl file %s because of errors %s", filePath, hclErrors)
	}
	return false, hclfile, nil
}

func ParseHclFileForAwsIamBlocks(hclfile *hclwrite.File) []*hclwrite.Block {
	var blocks []*hclwrite.Block
	for _, block := range hclfile.Body().Blocks() {
		if block.Type() == "resource" && IsBlockAwsIamRole(block) {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func IsBlockAwsIamRole(block *hclwrite.Block) bool {
	return len(block.Labels()) > 0 && block.Labels()[0] == "aws_iam_role"
}
func IsBlockInlinePolicy(block *hclwrite.Block) bool {
	return block.Type() == "inline_policy"
}
