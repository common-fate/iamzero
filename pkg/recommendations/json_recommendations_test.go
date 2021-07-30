package recommendations

import (
	"html/template"
	"testing"
)

func buildExampleAdvisor() AdviceFactory {
	return GetJSONAdvice(JSONPolicyParams{
		Policy: []Statement{{
			Action:   []string{"s3:PutObject"},
			Resource: []string{"arn:aws:s3:::{{ .Bucket }}/{{ .Key }}"},
		}},
		Comment: "Allow PutObject access to the specific key",
	})
}

func buildSampleEvent() AWSEvent {
	return AWSEvent{
		Time: "",
		Data: AWSData{
			Service:   "s3",
			Region:    "ap-southeast-2",
			Operation: "PutObject",
			Parameters: map[string]interface{}{
				"Bucket": "test-bucket",
				"Key":    "sample-object",
			},
		},
		Identity: AWSIdentity{
			User:    "iamzero-test-role",
			Role:    "arn:aws:iam::123456789012:role/iamzero-test-role",
			Account: "123456789012",
		},
	}
}

func TestJSONRecommendationsWorks(t *testing.T) {
	advisor := buildExampleAdvisor()
	e := buildSampleEvent()

	_, err := advisor(e)
	if err != nil {
		t.Fatal(err)
	}
}

// We expect that the advisor will find the cloud resources associated with a captured action
// in this case we expect the resource to be "test-bucket/sample-object"
func TestJSONRecommendationsSetsResources(t *testing.T) {
	advisor := buildExampleAdvisor()
	e := buildSampleEvent()

	a, _ := advisor(e)
	resources := a.Details().Resources
	if len(resources) != 1 {
		t.Fatal("expected 1 resource to be found")
	}

	expected := "test-bucket/sample-object"
	name := resources[0].Name

	if name != expected {
		t.Fatalf("parsed resource name did not match expected, name=%s expected=%s", name, expected)
	}
}

func TestParseResourceFromTemplateWorks(t *testing.T) {
	resourceTmpl := "arn:aws:s3:::{{ .Bucket }}/{{ .Key }}"

	tmpl := template.Must(template.New("policy").Parse(resourceTmpl))
	vars := map[string]interface{}{
		"Bucket": "test-bucket",
		"Key":    "test-key",
	}

	resource, err := parseResourceFromTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}
	if resource != "test-bucket/test-key" {
		t.Fatalf("resouce not parsed as expected, resource=%s", resource)
	}
}

func TestParseResourceFromTemplateIgnoresAccountAndRegion(t *testing.T) {
	resourceTmpl := "arn:aws:dynamodb:{{ .Region }}:{{ .Account }}:table/{{ .Table }}/index/*"

	tmpl := template.Must(template.New("policy").Parse(resourceTmpl))
	vars := map[string]interface{}{
		"Account": "123456789012",
		"Region":  "ap-southeast-2",
		"Table":   "test-table",
	}

	resource, err := parseResourceFromTemplate(tmpl, vars)
	if err != nil {
		t.Fatal(err)
	}
	if resource != "test-table" {
		t.Fatalf("resouce not parsed as expected, resource=%s", resource)
	}
}

func TestParseResourceFromTemplateGivesErrorIfNoVariables(t *testing.T) {
	resourceTmpl := "arn:aws:dynamodb"

	tmpl := template.Must(template.New("policy").Parse(resourceTmpl))
	vars := map[string]interface{}{}

	_, err := parseResourceFromTemplate(tmpl, vars)
	if err == nil {
		t.Fatal("expected error")
	}
}
