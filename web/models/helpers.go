package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"strings"

	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/crowdmob/goamz/dynamodb"
	"github.com/convox/kernel/web/Godeps/_workspace/src/github.com/goamz/goamz/cloudformation"
)

func availabilityZones() ([]string, error) {
	res, err := EC2.DescribeAvailabilityZones(nil, nil)

	if err != nil {
		return nil, err
	}

	subnets := make([]string, len(res.AvailabilityZones))

	for i, zone := range res.AvailabilityZones {
		subnets[i] = zone.Name
	}

	return subnets, nil
}

func buildFormationTemplate(name, section string, object interface{}) (string, error) {
	tmpl, err := template.New(section).Funcs(templateHelpers()).ParseFiles(fmt.Sprintf("formation/%s.tmpl", name))

	if err != nil {
		return "", err
	}

	var formation bytes.Buffer

	err = tmpl.Execute(&formation, object)

	if err != nil {
		return "", err
	}

	return formation.String(), nil
}

func coalesce(att *dynamodb.Attribute, def string) string {
	if att != nil {
		return att.Value
	} else {
		return def
	}
}

func createStack(formation, name string, params map[string]string, tags map[string]string) error {
	sp := &cloudformation.CreateStackParams{
		StackName:    name,
		TemplateBody: formation,
	}

	for key, value := range params {
		sp.Parameters = append(sp.Parameters, cloudformation.Parameter{ParameterKey: key, ParameterValue: value})
	}

	for key, value := range tags {
		sp.Tags = append(sp.Tags, cloudformation.Tag{Key: key, Value: value})
	}

	_, err := CloudFormation.CreateStack(sp)

	return err
}

func divideSubnet(base string, num int) ([]string, error) {
	if num > 4 {
		return nil, fmt.Errorf("too many divisions")
	}

	div := make([]string, num)
	parts := strings.Split(base, ".")

	for i := 0; i < num; i++ {
		div[i] = fmt.Sprintf("%s.%s.%s.%d/27", parts[0], parts[1], parts[2], i*32)
	}

	return div, nil
}

func flattenTags(tags []cloudformation.Tag) map[string]string {
	f := make(map[string]string)

	for _, tag := range tags {
		f[tag.Key] = tag.Value
	}

	return f
}

var idAlphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}

func humanStatus(original string) string {
	switch original {
	case "":
		return "new"
	case "CREATE_IN_PROGRESS":
		return "creating"
	case "CREATE_COMPLETE":
		return "running"
	case "DELETE_FAILED":
		return "running"
	case "DELETE_IN_PROGRESS":
		return "deleting"
	case "ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "ROLLBACK_COMPLETE":
		return "failed"
	case "UPDATE_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS":
		return "updating"
	case "UPDATE_COMPLETE":
		return "running"
	case "UPDATE_ROLLBACK_IN_PROGRESS":
		return "rollback"
	case "UPDATE_ROLLBACK_COMPLETE":
		return "failed"
	default:
		fmt.Printf("unknown status: %s\n", original)
		return "unknown"
	}
}

func nextAvailableSubnet(vpc string) (string, error) {
	res, err := CloudFormation.DescribeStacks("", "")

	if err != nil {
		return "", err
	}

	available := make([]string, 254)

	for i := 1; i <= 254; i++ {
		available[i-1] = fmt.Sprintf("10.0.%d.0/24", i)
	}

	used := make([]string, 0)

	for _, stack := range res.Stacks {
		tags := stackTags(stack)
		if tags["type"] == "app" {
			used = append(used, tags["subnet"])
		}
	}

	for _, a := range available {
		found := false

		for _, u := range used {
			if a == u {
				found = true
				break
			}
		}

		if !found {
			return a, nil
		}
	}

	return "", fmt.Errorf("no available subnets")
}

func prettyJson(raw string) (string, error) {
	var parsed map[string]interface{}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", err
	}

	bp, err := json.MarshalIndent(parsed, "", "  ")

	if err != nil {
		return "", err
	}

	return string(bp), nil
}

func printLines(data string) {
	lines := strings.Split(data, "\n")

	for i, line := range lines {
		fmt.Printf("%d: %s\n", i, line)
	}
}

func stackParameters(stack cloudformation.Stack) map[string]string {
	parameters := make(map[string]string)

	for _, parameter := range stack.Parameters {
		parameters[parameter.ParameterKey] = parameter.ParameterValue
	}

	return parameters
}

func stackTags(stack cloudformation.Stack) map[string]string {
	tags := make(map[string]string)

	for _, tag := range stack.Tags {
		tags[tag.Key] = tag.Value
	}

	return tags
}

func stackOutputs(stack cloudformation.Stack) map[string]string {
	outputs := make(map[string]string)

	for _, output := range stack.Outputs {
		outputs[output.OutputKey] = output.OutputValue
	}

	return outputs
}

func templateHelpers() template.FuncMap {
	return template.FuncMap{
		"array": func(ss []string) template.HTML {
			as := make([]string, len(ss))
			for i, s := range ss {
				as[i] = fmt.Sprintf("%q", s)
			}
			return template.HTML(strings.Join(as, ", "))
		},
		"ports": func(nn []int) template.HTML {
			as := make([]string, len(nn))
			for i, n := range nn {
				as[i] = fmt.Sprintf("%d", n)
			}
			return template.HTML(strings.Join(as, ","))
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"upper": func(s string) string {
			return upperName(s)
		},
	}
}

func upperName(name string) string {
	return strings.ToUpper(name[0:1]) + name[1:]
}