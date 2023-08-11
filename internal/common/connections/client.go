package connections

import (
	"context"
	"fmt"
)

type Client struct {
	token              string
	userAgent          string
	integrationsURL    string
	hostedExportersURL string
}

func New(token string, userAgent string, integrationsURL string, hostedExportersURL string) (*Client, error) {
	return &Client{
		token:              token,
		userAgent:          userAgent,
		integrationsURL:    integrationsURL,
		hostedExportersURL: hostedExportersURL,
	}, nil
}

// AWSConnection represents a connection to a user's AWS account.
type AWSConnection struct {
	// Name is a human-friendly identifier of the connection.
	Name string `json:"name"`

	// AWS is the ID of the connected AWS account.
	AWSAccountID string `json:"aws_account_id"`

	// RoleARN is the ARN of the associated IAM role.
	RoleARN string `json:"role_arn"`

	// Regions is the list of AWS regions that the connections can be used with
	Regions []string `json:"regions"`
}

func (client *Client) CreateAWSConnection(context context.Context, stackID string, connection AWSConnection) error {
	fmt.Printf("Create %s, %s\n", stackID, connection.Name)
	return nil
}

func (client *Client) GetAWSConnection(ctx context.Context, stackID string, name string) (*AWSConnection, error) {
	fmt.Printf("Get %s, %s\n", stackID, name)
	return &AWSConnection{}, nil
}

func (client *Client) UpdateAWSConnection(ctx context.Context, stackID string, connection AWSConnection) error {
	fmt.Printf("Update %s, %s\n", stackID, connection.Name)
	return nil
}

func (client *Client) DeleteAWSConnection(ctx context.Context, stackID string, name string) error {
	fmt.Printf("Delete %s, %s\n", stackID, name)
	return nil
}
