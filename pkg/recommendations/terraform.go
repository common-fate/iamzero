package recommendations

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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
	Arn  string `json:"arn"`
	Name string `json:"name"`
}

type PendingChanges struct {
	FilePath    string
	FileContent []byte
}

func ApplyTerraformFinding(finding *TerraformFinding) ([]PendingChanges, error) {

	/*
		We ideally want to lazily load any files that we need
		first we open and parse the statefile
		then we can scan the statefile to find teh resources that we need to interact with
		the statefile tells us the module name and the file path of all the resources that we need to update
		open and parse these files as needed
	*/

	/*
		1. remove existing inline policies
		2. add new inline policies
		3. cleanup any manage policy attachments
	*/

	// READ THE TERRAFORM FILE AND PARSE IT FOR IAM BLOCKS, CURRENTLY ONLY SEARCHES A SINGLE MAIN.TF FILE

	stateFile, err := parseTerraformState()
	if err != nil {
		fmt.Println(err)
		return []PendingChanges{}, err
	}

	// awsIamBlocks := ParseHclFileForAwsIamBlocks(hclFile)
	stateFileResource, terraformFilePath, _ := stateFile.findResourceInStateFileByArn(finding.Role)
	hclFile, err := OpenAndParseHclFile(terraformFilePath)
	if err != nil {
		return []PendingChanges{}, err
	}

	awsIamBlock := FindIamRoleBlockByModuleAddress(hclFile.Body().Blocks(), stateFileResource.Type+"."+stateFileResource.Name) //the stateFileResource.Address property contains the full paths from teh root which is not what I want

	if awsIamBlock != nil {
		return ApplyFindingToBlock(terraformFilePath, hclFile, awsIamBlock, finding, &stateFile)
	}
	// if we don't find teh matching role, either something went wrong in our code, or the statefile doesn't match the source code.
	// the user probably needs to run `terraform plan` again
	return []PendingChanges{}, fmt.Errorf("an error occurred finding the matching iam role in your terraform project, your state file may be outdated")

}

func FindIamRoleBlockByModuleAddress(blocks []*hclwrite.Block, moduleAddress string) *hclwrite.Block {
	//filters the blocks to find the target block by matching the address
	// and address looks like "aws_iam_role.iamzero-overprivileged-role"

	// for multiple files, this address will refer to the file by is path e.g "modules.ec2.aws_iam_role.role-name"
	for _, block := range blocks {
		if IsBlockAwsIamRole(block) {
			if resourceLabelToString(block.Labels()) == moduleAddress {
				return block
			}
		}
	}
	return nil
}

func resourceLabelToString(labels []string) string {
	return strings.Join(labels[:], ".")
}

func ApplyFindingToBlock(filePath string, hclFile *hclwrite.File, awsIamBlock *hclwrite.Block, finding *TerraformFinding, stateFile *StateFile) ([]PendingChanges, error) {
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
			a, _, err := stateFile.findResourceInStateFileByArn(*statement.Resources[0].ARN)
			if err == nil {
				// if we find a matching resource, then use the path to that resource
				arn = a.Address + ".arn"
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
		fmt.Println("removing block " + blockToRemove.Type() + strconv.FormatBool(awsIamBlock.Body().RemoveBlock(blockToRemove)))
	}
	// add the new blocks(inline policies) to this role
	for _, blockToAdd := range newInlinePolicies {
		awsIamBlock.Body().AppendBlock(blockToAdd)
	}
	return []PendingChanges{{FileContent: hclwrite.Format(hclFile.Bytes()), FilePath: filePath}}, nil

}

func (s StateFile) findResourceInStateFileByArn(arn string) (StateFileResource, string, error) {
	return s.findResourceInStateFileBase(arn, func(s StateFileResource) string {
		return s.Values.Arn
	})
}
func (s StateFile) findResourceInStateFileByName(name string) (StateFileResource, string, error) {
	return s.findResourceInStateFileBase(name, func(s StateFileResource) string {
		return s.Values.Name
	})
}

func (s StateFile) findResourceInStateFileBase(value string, attributefn func(s StateFileResource) string) (StateFileResource, string, error) {
	/*
		I added this to allow matching of any of the properties of the resource by passing in a function to select the attribute
	*/
	terraformFilePath := "main.tf"
	for _, r := range s.Values.RootModule.Resources {
		if attributefn(r) == value {
			return r, terraformFilePath, nil
		}
	}
	for _, module := range s.Values.RootModule.ChildModules {
		for _, r := range module.Resources {
			if attributefn(r) == value {

				terraformFilePath = filepath.Join("modules", filepath.Join(strings.Split(module.Address, ".")[1:]...), terraformFilePath)
				return r, terraformFilePath, nil
			}
		}

	}
	return StateFileResource{}, "", fmt.Errorf("not found")
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

func WriteFile(readFilePath string, hclFile *hclwrite.File, writeFilePath string) error {

	// heavily inspired by the YOR library
	tempFile, err := ioutil.TempFile(filepath.Dir(readFilePath), "temp.*.tf")
	defer func() {
		_ = os.Remove(tempFile.Name())
	}()
	if err != nil {
		return err
	}
	fd, err := os.OpenFile(tempFile.Name(), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = hclFile.WriteTo(fd)
	if err != nil {
		return err
	}

	// We format the file bytes before writing the file
	err = os.WriteFile(writeFilePath, hclwrite.Format(hclFile.Bytes()), 0600)
	if err != nil {
		return fmt.Errorf("failed to write HCL file %s, %s", readFilePath, err.Error())
	}

	return nil
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

func OpenAndParseHclFile(filePath string) (*hclwrite.File, error) {
	// read file bytes
	src, err := ioutil.ReadFile(filePath)
	if err != nil {
		return hclwrite.NewEmptyFile(), fmt.Errorf("failed to read file %s because %s", filePath, err)
	}
	hclfile, diagnostics := hclwrite.ParseConfig(src, filePath, hcl.InitialPos)
	if diagnostics != nil && diagnostics.HasErrors() {
		hclErrors := diagnostics.Errs()
		return hclwrite.NewEmptyFile(), fmt.Errorf("failed to parse hcl file %s because of errors %s", filePath, hclErrors)
	}
	return hclfile, nil
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
	return len(block.Labels()) > 0 && strings.Contains(block.Labels()[0], "aws_iam_role")
}
func IsBlockInlinePolicy(block *hclwrite.Block) bool {
	return block.Type() == "inline_policy"
}
