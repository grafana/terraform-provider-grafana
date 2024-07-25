package provider

import (
	"os"
	"testing"
)

func TestCreateTempFileIfLiteral(t *testing.T) {
	t.Run("Test with empty string value returns empty and does not create temp file", func(t *testing.T) {
		path, tempFile, err := createTempFileIfLiteral("")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if tempFile {
			t.Fatalf("Expected tempFile to be false, got %v", tempFile)
		}
		if path != "" {
			t.Fatalf("Expected empty path, got %v", path)
		}
	})
	t.Run("Test file path returns given path and does not create a temp file", func(t *testing.T) {
		// Create a temporary file to simulate an existing file
		tmp, err := os.CreateTemp("", "existing-file")
		if err != nil {
			t.Fatalf("Failed to create file for test, error: %v", err)
		}
		defer os.Remove(tmp.Name())

		path, tempFile, err := createTempFileIfLiteral(tmp.Name())
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if tempFile {
			t.Fatalf("Expected tempFile to be false, got %v", tempFile)
		}
		if path != tmp.Name() {
			t.Fatalf("Expected path to be '%s', got '%s'", tmp.Name(), path)
		}
	})

	t.Run("Test with short literal creates temp file and path", func(t *testing.T) {
		caCert := "certTest"

		path, tempFile, err := createTempFileIfLiteral(caCert)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !tempFile {
			t.Fatalf("Expected tempFile to be true, got %v", tempFile)
		}
		if path == "" {
			t.Fatalf("Expected a file path, got an empty string")
		}

		// Validate the file was created and has the correct content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Expected to read the file, got %v", err)
		}
		if string(content) != caCert {
			t.Fatalf("Expected file content to be '%s', got '%s'", caCert, string(content))
		}

		// Clean up the temporary file
		if err := os.Remove(path); err != nil {
			t.Fatalf("Expected to delete the file, got %v", err)
		}
	})

	t.Run("Test with a certificate literal creates temp file and path", func(t *testing.T) {
		caCert := ` -----BEGIN CERTIFICATE-----
	MIIDXTCCAkWgAwIBAgIJAMW9UJtz1MoNMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
	BAYTAkFVMRMwEQYDVQQIDApxdWVlbnNsYW5kMRAwDgYDVQQHDAdicmlzYmFuZTEN
	MAsGA1UECgwEVGVzdDAeFw0xODA2MTAwNzU1NDJaFw0xOTA2MTAwNzU1NDJaMEUx
	CzAJBgNVBAYTAkFVMRMwEQYDVQQIDApxdWVlbnNsYW5kMRAwDgYDVQQHDAdicmlz
	YmFuZTENMAsGA1UECgwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
	ggEBAK1lpt+lPZJbG7yMYYWzjk8FwGbM3vlUJlC2aQHJ18T2aTtsOaZC1deKtwGR
	qBZMyel3hG0XayZmFQO2DAnOScgn4j+jPEFLWswg+U4MgH80+PA4wHzm+E0v68qD
	S+cA9If1D2I0gtT6jKPm3WYwZ/r0GUn8/JjgiIhCZGfXArH39V2D2KNhJ3W0b7T6
	isfbsHvSKWs/49q8w5J/yN8GOh/n/rThBfhM3FQ2eDdVR1QfvvX5KT69aXhtJlD9
	Z5H8z9DnD8BZxBrzE5hEO74KK13CvAFeKbVp7KvXf6NOy4W31lUd6lmzZ+lR+IxO
	NHElgJoaJ2F2y4XcFXY1cQFhKjkCAwEAAaNQME4wHQYDVR0OBBYEFLPkkSMxs/PR
	1E7VwDhRu5DTHwrNMB8GA1UdIwQYMBaAFLPkkSMxs/PR1E7VwDhRu5DTHwrNMAwG
	A1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAK1lpt+lPZJbG7yMYYWzjk8F
	wGbM3vlUJlC2aQHJ18T2aTtsOaZC1deKtwGRqBZMyel3hG0XayZmFQO2DAnOScgn
	4j+jPEFLWswg+U4MgH80+PA4wHzm+E0v68qDS+cA9If1D2I0gtT6jKPm3WYwZ/r0
	GUn8/JjgiIhCZGfXArH39V2D2KNhJ3W0b7T6isfbsHvSKWs/49q8w5J/yN8GOh/n
	/rThBfhM3FQ2eDdVR1QfvvX5KT69aXhtJlD9Z5H8z9DnD8BZxBrzE5hEO74KK13C
	vAFeKbVp7KvXf6NOy4W31lUd6lmzZ+lR+IxONHElgJoaJ2F2y4XcFXY1cQFhKjkC
	AwEAAaNQME4wHQYDVR0OBBYEFLPkkSMxs/PR1E7VwDhRu5DTHwrNMB8GA1UdIwQY
	MBaAFLPkkSMxs/PR1E7VwDhRu5DTHwrNMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcN
	AQELBQADggEBAK1lpt+lPZJbG7yMYYWzjk8FwGbM3vlUJlC2aQHJ18T2aTtsOaZC
	1deKtwGRqBZMyel3hG0XayZmFQO2DAnOScgn4j+jPEFLWswg+U4MgH80+PA4wHzm
	+E0v68qDS+cA9If1D2I0gtT6jKPm3WYwZ/r0GUn8/JjgiIhCZGfXArH39V2D2KNh
	J3W0b7T6isfbsHvSKWs/49q8w5J/yN8GOh/n/rThBfhM3FQ2eDdVR1QfvvX5KT69
	aXhtJlD9Z5H8z9DnD8BZxBrzE5hEO74KK13CvAFeKbVp7KvXf6NOy4W31lUd6lmz
	Z+lR+IxONHElgJoaJ2F2y4XcFXY1cQFhKjkCAwEAAaNQME4wHQYDVR0OBBYEFLPk
	kSMxs/PR1E7VwDhRu5DTHwrNMB8GA1UdIwQYMBaAFLPkkSMxs/PR1E7VwDhRu5DT
	HwrNMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAK1lpt+lPZJbG7yM
	YYWzjk8FwGbM3vlUJl=
    -----END CERTIFICATE-----`

		path, tempFile, err := createTempFileIfLiteral(caCert)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !tempFile {
			t.Fatalf("Expected tempFile to be true, got %v", tempFile)
		}
		if path == "" {
			t.Fatalf("Expected a file path, got an empty string")
		}

		// Check if the file exists and has the correct content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Expected to read the file, got %v", err)
		}
		if string(content) != caCert {
			t.Fatalf("Expected file content to be '%s', got '%s'", caCert, string(content))
		}

		// Clean up the temporary file
		if err := os.Remove(path); err != nil {
			t.Fatalf("Expected to delete the file, got %v", err)
		}
	})
}
