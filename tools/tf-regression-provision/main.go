// Command tf-regression-provision creates or destroys the throwaway PingOne
// environment used by the provisioned-environment E2E test tier.
//
// Everything - the environment itself (pingone_environment) and every
// fixture resource inside it - is defined as Terraform in
// terraform-test-data/root, authenticated with a single org-admin
// credential. This tool is a thin wrapper around `terraform apply`/`destroy`
// plus output parsing; it holds no PingOne API logic of its own.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const tfvarsFileName = "terraform.tfvars.json"

// createResult is printed to stdout as JSON on a successful `create` so the
// caller (shell script or CI step) can pick up the credentials to use for
// the subsequent export/compare phase. It is deliberately never written to
// a report file - callers must mask/consume it directly, never let it land
// on disk in a CI artifact.
//
// AuthEnvironmentID and TargetEnvironmentID are deliberately distinct: the
// org-admin credential's OAuth token is acquired against its own home
// environment (AuthEnvironmentID), but the resources being exported live in
// the newly created throwaway environment (TargetEnvironmentID). This
// mirrors internal/platform/pingone.NewFromCredentials's workerEnvID vs
// exportEnvID split.
type createResult struct {
	AuthEnvironmentID   string `json:"auth_environment_id"`
	TargetEnvironmentID string `json:"target_environment_id"`
	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	RegionCode          string `json:"region_code"`
}

type orgAdminConfig struct {
	clientID      string
	clientSecret  string
	environmentID string
	regionCode    string
	licenseID     string
}

func main() {
	action := flag.String("action", "", "create or destroy")
	tfDir := flag.String("terraform-dir", "terraform-test-data/root", "path to the Terraform root module to apply/destroy")
	namePrefix := flag.String("name-prefix", "pingcli-terraformer-e2e", "prefix for the created environment name")
	flag.Parse()

	cfg := orgAdminConfig{
		clientID:      os.Getenv("PINGCLI_PINGONE_ORGADMIN_CLIENT_ID"),
		clientSecret:  os.Getenv("PINGCLI_PINGONE_ORGADMIN_CLIENT_SECRET"),
		environmentID: os.Getenv("PINGCLI_PINGONE_ORGADMIN_ENVIRONMENT_ID"),
		regionCode:    os.Getenv("PINGCLI_PINGONE_ORGADMIN_REGION_CODE"),
		licenseID:     os.Getenv("PINGCLI_PINGONE_ORGADMIN_LICENSE_ID"),
	}

	switch *action {
	case "create":
		if err := runCreate(cfg, *tfDir, *namePrefix); err != nil {
			log.Fatalf("create failed: %v", err)
		}
	case "destroy":
		if err := runDestroy(*tfDir); err != nil {
			log.Fatalf("destroy failed: %v", err)
		}
	default:
		fmt.Fprintln(os.Stderr, "Usage: tf-regression-provision --action=create|destroy [--terraform-dir <path>]")
		os.Exit(2)
	}
}

func runCreate(cfg orgAdminConfig, tfDir, namePrefix string) error {
	envName := fmt.Sprintf("%s-%d", namePrefix, time.Now().Unix())
	if err := writeTFVars(tfDir, cfg, envName); err != nil {
		return fmt.Errorf("write %s: %w", tfvarsFileName, err)
	}

	if err := runTerraform(tfDir, "init", "-input=false"); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	if err := runTerraform(tfDir, "apply", "-auto-approve", "-input=false"); err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}

	targetEnvID, err := terraformOutput(tfDir, "environment_id")
	if err != nil {
		return fmt.Errorf("read environment_id output: %w", err)
	}

	result := createResult{
		AuthEnvironmentID:   cfg.environmentID,
		TargetEnvironmentID: targetEnvID,
		ClientID:            cfg.clientID,
		ClientSecret:        cfg.clientSecret,
		RegionCode:          cfg.regionCode,
	}
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(result)
}

func runDestroy(tfDir string) error {
	if err := runTerraform(tfDir, "init", "-input=false"); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	if err := runTerraform(tfDir, "destroy", "-auto-approve", "-input=false"); err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}
	if err := os.Remove(filepath.Join(tfDir, tfvarsFileName)); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: failed to remove %s: %v", tfvarsFileName, err)
	}
	return nil
}

// writeTFVars persists the org-admin credentials and the desired
// environment name as a terraform.tfvars.json file in tfDir, which
// Terraform loads automatically on every subsequent init/apply/destroy in
// that directory - including a later `destroy` invocation, which runs as a
// separate process with no access to create's in-memory values.
func writeTFVars(tfDir string, cfg orgAdminConfig, envName string) error {
	vars := map[string]string{
		"org_admin_environment_id": cfg.environmentID,
		"org_admin_client_id":      cfg.clientID,
		"org_admin_client_secret":  cfg.clientSecret,
		"region_code":              cfg.regionCode,
		"license_id":               cfg.licenseID,
		"environment_name":         envName,
	}
	data, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(tfDir, tfvarsFileName), data, 0600)
}

func terraformOutput(tfDir, name string) (string, error) {
	cmd := exec.Command("terraform", "output", "-raw", name)
	cmd.Dir = tfDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// runTerraform sends the child's stdout to our stderr, not our stdout:
// `create` prints exactly one JSON line to stdout as its machine-readable
// result, and Terraform's own progress output must never share that stream.
func runTerraform(dir string, args ...string) error {
	cmd := exec.Command("terraform", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
