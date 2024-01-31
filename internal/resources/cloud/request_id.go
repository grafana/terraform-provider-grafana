package cloud

import "github.com/hashicorp/go-uuid"

func clientRequestID() string {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return ""
	}
	return "tf-" + uuid
}
