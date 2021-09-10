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
	Role            TerraformRole             `json:"role"`
	Recommendations []TerraformRecommendation `json:"recommendations"`
}

// TerraformRole is a reference to a user or role defined in Terraform
type TerraformRole struct {
	Name string `json:"name"`
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
	Type      string  `json:"type"`
	ARN       *string `json:"arn,omitempty"`
}

type StateFile struct {
	Values StateFileValues `json:"values"`
}

type StateFileValues struct {
	RootModule RootModule `json:"root_module"`
}
type RootModule struct {
	Resources []StateFileResource `json:"resources"`
}

type StateFileResource struct {
	Type    string             `json:"type"`
	Name    string             `json:"name"`
	Address string             `json:"address"`
	Values  StateFileAttribute `json:"values"`
}

type StateFileAttribute struct {
	Arn string `json:"arn"`
}

func ApplyTerraformFinding(finding *TerraformFinding) []byte {

	// READ THE TERRAFORM FILE AND PARSE IT FOR IAM BLOCKS, CURRENTLY ONLY SEARCHES A SINGLE MAIN.TF FILE
	hclFile, _ := OpenAndParseHclFile("./main.tf")
	stateFile, err := parseTerraformState()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	awsIamBlocks := ParseHclFileForAwsIamBlocks(hclFile)
	return ApplyFindingToBlocks(hclFile, awsIamBlocks, finding, &stateFile)
}

func ApplyFindingToBlocks(hclFile *hclwrite.File, awsIamBlocks []*hclwrite.Block, finding *TerraformFinding, stateFile *StateFile) []byte {
	for _, block := range awsIamBlocks {
		if IsBlockAwsIamRole(block) && StringCompareAttributeValue(block.Body().GetAttribute("name"), finding.Role.Name) {
			newInlinePolicies := []*hclwrite.Block{}
			existingInlinePoliciesToRemove := []*hclwrite.Block{}
			for _, nestedBlock := range block.Body().Blocks() {
				if IsBlockInlinePolicy(nestedBlock) {
					existingInlinePoliciesToRemove = append(existingInlinePoliciesToRemove, nestedBlock)
				}
			}
			for _, reccomendation := range finding.Recommendations {
				for i, statement := range reccomendation.Statements {
					newBlock := hclwrite.NewBlock("inline_policy", nil)
					actionsJson, _ := json.Marshal(statement.Actions)
					arn := ""
					a, err := stateFile.findResourceInStateFile(*statement.Resources[0].ARN)
					if err == nil {
						// if we find a matching resource, then use the path to that resource
						arn = a.Address + ".arn"
					} else {
						// add quotes around it to make it valid for our use, maybe a json stringify equivalent would be good here to add the quotes robustly
						arn = fmt.Sprintf(`"%s"`, *statement.Resources[0].ARN)
						fmt.Printf("Failed to find matching resource in state file %s %s\n", arn, stateFile)
					}

					setInlinePolicyIamPolicy(newBlock, string(actionsJson), arn, "iamzero-generated-iam-policy-"+fmt.Sprint(i))
					newInlinePolicies = append(newInlinePolicies, newBlock)
				}
			}
			// some logic probably works out whether all existing policies need to go
			for _, blockToRemove := range existingInlinePoliciesToRemove {
				fmt.Println("removing block " + blockToRemove.Type() + strconv.FormatBool(block.Body().RemoveBlock(blockToRemove)))

			}
			// add the new blocks(inline policies) to this role
			for _, blockToAdd := range newInlinePolicies {
				block.Body().AppendBlock(blockToAdd)
			}
		}
	}
	return hclwrite.Format(hclFile.Bytes())
}

func (s StateFile) findResourceInStateFile(arn string) (StateFileResource, error) {
	for _, r := range s.Values.RootModule.Resources {
		if r.Values.Arn == arn {
			return r, nil
		}
	}
	return StateFileResource{}, fmt.Errorf("not found")
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
	// A valid name will include "" quotes so these are added around the input string being compared
	return strings.Trim(string(attribute.Expr().BuildTokens(hclwrite.Tokens{}).Bytes()), " ") == fmt.Sprintf(`"%s"`, compareTo)
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
