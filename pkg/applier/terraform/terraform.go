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
	FilePath         string
	FilePathFromRoot string
	ParentAddress    string
}

type FileHandler struct {
	HclFiles map[string]*hclwrite.File
}

type AwsIamBlock struct {
	*hclwrite.Block
}

type TerraformIAMPolicyApplier struct {
	AWSIAMPolicyApplier applier.AWSIAMPolicyApplier
	Finding             *TerraformFinding
	FileHandler         *FileHandler
	StateFile           *StateFile
}

var MAIN_TERRAFORM_FILE = "main.tf"

func (t *TerraformIAMPolicyApplier) GetProjectName() string { return "Terraform" }

func (t *TerraformIAMPolicyApplier) Init() error {
	// Init File handler to manage reading and writing
	t.FileHandler = &FileHandler{HclFiles: make(map[string]*hclwrite.File)}

	// load the statefile
	stateFile, err := t.parseTerraformState()
	if err != nil {
		return err
	}
	t.StateFile = stateFile
	return nil
}

func (t *TerraformIAMPolicyApplier) Detect() bool {
	_, err := os.Stat(t.getRootFilePath())
	return err == nil
}

func (t *TerraformIAMPolicyApplier) CalculateFinding(policy *recommendations.Policy, actions []recommendations.AWSAction) {
	t.calculateTerraformFinding(policy, actions)
}
func (t *TerraformIAMPolicyApplier) Plan() (*applier.PendingChanges, error) {
	return t.PlanTerraformFinding()
}

func (t *TerraformIAMPolicyApplier) Apply(changes *applier.PendingChanges) error {
	// Writes the changes to the files
	for _, change := range *changes {
		err := ioutil.WriteFile(change.Path, []byte(change.Contents), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *TerraformIAMPolicyApplier) getRootFilePath() string {
	return path.Join(t.AWSIAMPolicyApplier.ProjectPath, MAIN_TERRAFORM_FILE)
}

func (t *TerraformIAMPolicyApplier) calculateTerraformFinding(policy *recommendations.Policy, actions []recommendations.AWSAction) {

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

func (t *TerraformIAMPolicyApplier) FindAndRemovePolicyAttachmentsForRole(stateFileRole StateFileResource) {
	attachments := t.FindPolicyAttachmentsInStateFileByRoleName(stateFileRole.Values.Name)
	for key, element := range attachments {
		// for each terraform file, update all the modules
		hclFile, _ := t.FileHandler.OpenFile(key, false)
		for _, stateFileResource := range element {
			policyAttachmentBlock := FindBlockByModuleAddress(hclFile.Body().Blocks(), stateFileResource.Type+"."+stateFileResource.Name)
			RemovePolicyAttachmentRole(policyAttachmentBlock, stateFileRole)
		}
	}

}

func (t *TerraformIAMPolicyApplier) PendingChanges() *applier.PendingChanges {
	/*
		return the formatted bytes for each open file
	*/
	pc := applier.PendingChanges{}
	for key, val := range t.FileHandler.HclFiles {
		pc = append(pc, applier.PendingChange{Path: key, Contents: string(hclwrite.Format(val.Bytes()))})
	}
	return &pc
}

func (t *TerraformIAMPolicyApplier) PlanTerraformFinding() (*applier.PendingChanges, error) {
	iamRoleStateFileResource, err := t.FindResourceInStateFileByArn(t.Finding.Role)
	if err != nil {
		return nil, err
	}
	hclFile, err := t.FileHandler.OpenFile(iamRoleStateFileResource.FilePath, false)
	if err != nil {
		return nil, err
	}
	awsIamBlock := FindIamRoleBlockByModuleAddress(hclFile.Body().Blocks(), iamRoleStateFileResource.Type+"."+iamRoleStateFileResource.Name)
	if awsIamBlock != nil {
		// Remove any managed policy attachments for this role
		t.FindAndRemovePolicyAttachmentsForRole(iamRoleStateFileResource.StateFileResource)
		rootHclFile, err := t.FileHandler.OpenFile(t.getRootFilePath(), false)
		if err != nil {
			return nil, err
		}
		// Apply the finding by appending inline policies to the role
		err = t.ApplyFindingToBlock(awsIamBlock, iamRoleStateFileResource, rootHclFile)
		if err != nil {
			return nil, err
		}
		return t.PendingChanges(), nil
	}
	// If we don't find the matching role, either something went wrong in our code, or the statefile doesn't match the source code.
	// the user probably needs to run `terraform plan` again
	return nil, fmt.Errorf("an error occurred finding the matching iam role in your terraform project, your state file may be outdated, try running 'terraform plan'")

}
func (t *TerraformIAMPolicyApplier) ApplyFindingToBlock(awsIamBlock *AwsIamBlock, iamRoleStateFileResource StateFileResourceBlock, rootHclFile *hclwrite.File) error {
	newInlinePolicies := []*hclwrite.Block{}
	existingInlinePoliciesToRemove := []*hclwrite.Block{}
	for _, nestedBlock := range awsIamBlock.Body().Blocks() {
		if IsBlockInlinePolicy(nestedBlock) {
			existingInlinePoliciesToRemove = append(existingInlinePoliciesToRemove, nestedBlock)
		}
	}

	for _, reccomendation := range t.Finding.Recommendations {
		for i, statement := range reccomendation.Statements {
			newBlock := hclwrite.NewBlock("inline_policy", nil)
			actionsJson, _ := json.Marshal(statement.Actions)
			arn := ""

			// The ARN from the recommendation can contain "/*" on the end for an s3 bucket, to look this up in
			// @TODO probably need to verify whether we could get specific things here(as in specific objects in a bucket)

			splitArn := statement.Resources[0].SplitArn()
			awsResource, err := t.FindResourceInStateFileByArn(splitArn[0])
			if err == nil {
				/*
					THE BELOW SCENARIOS ONLY SUPPORT A FLAT PROJECT STRUCTURE WHERE THERE IS ONLY 1 LEVEL OF MODULE ABSTRACTION

					MAIN.TF
						->MODULES
							->EC2
								MAIN.TF

				*/
				if awsResource.ParentAddress == iamRoleStateFileResource.ParentAddress {
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
					outputsFile, err := t.FileHandler.OpenFile(outputsFilePath, true)
					if err != nil {
						return err
					}

					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.Name, "arn"), "IAMZero generated output for resource", strings.Trim(awsResource.Address, awsResource.ParentAddress+".")+".arn")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), awsResource.FilePathFromRoot)
					resourcePathInRootModule := ""
					if moduleDefinitionInRoot != nil {

						resourcePathInRootModule = strings.Join([]string{"module", moduleDefinitionInRoot.Labels()[0], outputName}, ".")
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", awsResource.FilePathFromRoot)
					}

					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(iamRoleStateFileResource.FilePath), "variables.tf")
					variablesFile, err := t.FileHandler.OpenFile(variableFilePath, true)
					if err != nil {
						return err
					}

					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.Name, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot = FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), iamRoleStateFileResource.FilePathFromRoot)

					if moduleDefinitionInRoot != nil {
						AddInputToModuleDeclaration(moduleDefinitionInRoot, variableName, resourcePathInRootModule)
						arn = "var." + variableName
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", iamRoleStateFileResource.FilePathFromRoot)
					}

				} else if !awsResource.IsInRoot() && iamRoleStateFileResource.IsInRoot() {

					// role is in root,  resource is in another file
					// create an output from the resource definition
					// refer to it in the root inline policy

					outputsFilePath := filepath.Join(filepath.Dir(awsResource.FilePath), "outputs.tf")
					outputsFile, err := t.FileHandler.OpenFile(outputsFilePath, true)
					if err != nil {
						return err
					}
					outputName := AppendOutputBlockIfNotExist(outputsFile.Body(), GenerateOutputName(awsResource.Name, "arn"), "IAMZero generated output for resource", strings.Trim(awsResource.Address, awsResource.ParentAddress+".")+".arn")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), awsResource.FilePathFromRoot)
					if moduleDefinitionInRoot != nil {

						arn = strings.Join([]string{"module", moduleDefinitionInRoot.Labels()[0], outputName}, ".")
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", awsResource.FilePathFromRoot)
					}

				} else if awsResource.IsInRoot() && !iamRoleStateFileResource.IsInRoot() {
					// resource is in root, role is in another file
					// create variable for role module
					// add variable value in root module declaration

					// add variable for a resource
					variableFilePath := filepath.Join(filepath.Dir(iamRoleStateFileResource.FilePath), "variables.tf")
					variablesFile, err := t.FileHandler.OpenFile(variableFilePath, true)
					if err != nil {
						return err
					}
					variableName := AppendVariableBlockIfNotExist(variablesFile.Body(), GenerateVariableName(awsResource.Name, "arn"), "IAMZero generated variable for resource")

					moduleDefinitionInRoot := FindModuleBlockBySourcePath(rootHclFile.Body().Blocks(), iamRoleStateFileResource.FilePathFromRoot)
					if moduleDefinitionInRoot != nil {
						AddInputToModuleDeclaration(moduleDefinitionInRoot, variableName, awsResource.Address+".arn")
						arn = "var." + variableName
					} else {
						return fmt.Errorf("failed to find module block in file Root file for :%s", iamRoleStateFileResource.FilePathFromRoot)
					}
				} else {
					// add quotes around it to make it valid for our use, maybe a json stringify equivalent would be good here to add the quotes robustly
					arn = fmt.Sprintf(`"%s"`, *statement.Resources[0].ARN)
					t.AWSIAMPolicyApplier.Logger.Warnf("Resource with ARN(%s) is declared more than 1 level into the project tree, this is not yet supported. Using ARN directly\n", arn, t.StateFile)
				}

			} else {
				// add quotes around it to make it valid for our use, maybe a json stringify equivalent would be good here to add the quotes robustly
				arn = fmt.Sprintf(`"%s"`, *statement.Resources[0].ARN)
				t.AWSIAMPolicyApplier.Logger.Warnf("Failed to find matching resource in state file for ARN(%s) in (%s) using ARN reference directly\n", arn, t.StateFile)
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

func (t *TerraformIAMPolicyApplier) FindResourceInStateFileByArn(arn string) (StateFileResourceBlock, error) {
	return t.FindResourceInStateFileBase(arn, func(s StateFileResource) string {
		return s.Values.Arn
	})
}
func (t *TerraformIAMPolicyApplier) FindResourceInStateFileByName(name string) (StateFileResourceBlock, error) {
	return t.FindResourceInStateFileBase(name, func(s StateFileResource) string {
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
func (t *TerraformIAMPolicyApplier) FindPolicyAttachmentsInStateFileByRoleName(name string) map[string][]StateFileResource {
	terraformFilePath := t.getRootFilePath()
	resourceType := "aws_iam_policy_attachment"
	matches := make(map[string][]StateFileResource)

	// First checks if its in the root modules
	for _, r := range t.StateFile.Values.RootModule.Resources {
		if r.Type == resourceType && sliceContains(r.Values.Roles, name) {
			if matches[terraformFilePath] == nil {
				matches[terraformFilePath] = []StateFileResource{}
			}
			matches[terraformFilePath] = append(matches[terraformFilePath], r)
		}
	}
	// Then will check if its in other modules below the root
	for _, module := range t.StateFile.Values.RootModule.ChildModules {
		for _, r := range module.Resources {
			if r.Type == resourceType && sliceContains(r.Values.Roles, name) {
				terraformFilePath = filepath.Join(t.AWSIAMPolicyApplier.ProjectPath, "modules", filepath.Join(strings.Split(module.Address, ".")[1:]...), MAIN_TERRAFORM_FILE)
				if matches[terraformFilePath] == nil {
					matches[terraformFilePath] = []StateFileResource{}
				}
				matches[terraformFilePath] = append(matches[terraformFilePath], r)
			}
		}

	}
	return matches
}

func (t *TerraformIAMPolicyApplier) FindResourceInStateFileBase(value string, attributefn func(s StateFileResource) string) (StateFileResourceBlock, error) {
	/*
		I added this to allow matching of any of the properties of the resource by passing in a function to select the attribute
	*/
	terraformFilePath := t.getRootFilePath()
	for _, r := range t.StateFile.Values.RootModule.Resources {
		if attributefn(r) == value {
			return StateFileResourceBlock{r, terraformFilePath, "./", "."}, nil
		}
	}
	for _, module := range t.StateFile.Values.RootModule.ChildModules {
		for _, r := range module.Resources {
			if attributefn(r) == value {
				terraformFilePathFromRoot := filepath.Join("modules", filepath.Join(strings.Split(module.Address, ".")[1:]...), MAIN_TERRAFORM_FILE)
				terraformFilePath = filepath.Join(t.AWSIAMPolicyApplier.ProjectPath, terraformFilePathFromRoot)

				return StateFileResourceBlock{r, terraformFilePath, terraformFilePathFromRoot, module.Address}, nil
			}
		}

	}
	return StateFileResourceBlock{StateFileResource{}, "", "", ""}, fmt.Errorf("could not find a matching resource in the state file for %s", value)
}
func (t *TerraformIAMPolicyApplier) parseTerraformState() (*StateFile, error) {
	// var terraformShow = []byte(`{"format_version":"0.1","terraform_version":"0.14.9","values":{"root_module":{"resources":[{"address":"aws_iam_role.iamzero-overprivileged-role","mode":"managed","type":"aws_iam_role","name":"iamzero-overprivileged-role","provider_name":"registry.terraform.io/hashicorp/aws","schema_version":0,"values":{"arn":"arn:aws:iam::312231318920:role/iamzero-tf-overprivileged-role","assume_role_policy":"{\"Version\":\"2012-10-17\",\"Statement\":[{\"Sid\":\"\",\"Effect\":\"Allow\",\"Principal\":{\"AWS\":\"arn:aws:iam::312231318920:root\"},\"Action\":\"sts:AssumeRole\"}]}","create_date":"2021-09-03T03:30:22Z","description":"","force_detach_policies":false,"id":"iamzero-tf-overprivileged-role","inline_policy":[{"name":"tf-example-policy","policy":"{\"Statement\":[{\"Action\":[\"*\"],\"Effect\":\"Allow\",\"Resource\":\"*\"}],\"Version\":\"2012-10-17\"}"}],"managed_policy_arns":[],"max_session_duration":3600,"name":"iamzero-tf-overprivileged-role","name_prefix":null,"path":"/","permissions_boundary":null,"tags":{},"tags_all":{},"unique_id":"AROAURMTP2WECJCRJBHTS"}},{"address":"aws_s3_bucket.iamzero-tf-example-bucket","mode":"managed","type":"aws_s3_bucket","name":"iamzero-tf-example-bucket","provider_name":"registry.terraform.io/hashicorp/aws","schema_version":0,"values":{"acceleration_status":"","acl":"private","arn":"arn:aws:s3:::iamzero-tf-example-bucket","bucket":"iamzero-tf-example-bucket","bucket_domain_name":"iamzero-tf-example-bucket.s3.amazonaws.com","bucket_prefix":null,"bucket_regional_domain_name":"iamzero-tf-example-bucket.s3.ap-southeast-2.amazonaws.com","cors_rule":[],"force_destroy":false,"grant":[],"hosted_zone_id":"Z1WCIGYICN2BYD","id":"iamzero-tf-example-bucket","lifecycle_rule":[],"logging":[],"object_lock_configuration":[],"policy":null,"region":"ap-southeast-2","replication_configuration":[],"request_payer":"BucketOwner","server_side_encryption_configuration":[],"tags":{},"tags_all":{},"versioning":[{"enabled":false,"mfa_delete":false}],"website":[],"website_domain":null,"website_endpoint":null}}]}}}`)

	// return MarshalStateFileToGo(terraformShow)
	// need to be in the root of the terraform repo for this to work
	// doing this gets the correct state for us,alternative would be to copy this code from teh terraform repo
	// JSON output via the -json option requires Terraform v0.12 or later.
	// https://www.terraform.io/docs/cli/commands/show.html
	_, err := exec.Command("terraform", "-chdir=./"+t.AWSIAMPolicyApplier.ProjectPath, "init").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to init terraform ' %s", err)
	}
	out, err := exec.Command("terraform", "-chdir=./"+t.AWSIAMPolicyApplier.ProjectPath, "show", "-json").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to load stateFile using 'terraform show -json' %s", err)
	}
	return MarshalStateFileToGo(out)
}

func MarshalStateFileToGo(stateFileBytes []byte) (*StateFile, error) {
	var stateFile StateFile
	err := json.Unmarshal(stateFileBytes, &stateFile)
	if err != nil {
		return nil, err
	}
	return &stateFile, nil
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
