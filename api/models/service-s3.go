package models

import (
	"fmt"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/cloudformation"
)

func (s *Service) CreateS3() (*cloudformation.CreateStackInput, error) {
	var input interface{}

	formation, err := buildTemplate(fmt.Sprintf("service/%s", s.Type), "service", input)

	if err != nil {
		return nil, err
	}

	req := &cloudformation.CreateStackInput{
		StackName:    aws.String(s.StackName()),
		TemplateBody: aws.String(formation),
	}

	return req, nil
}