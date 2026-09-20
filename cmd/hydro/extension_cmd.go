package main

import (
	"crypto/ed25519"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/extensionpkg"
)

type extensionStringList []string

func (v *extensionStringList) String() string { return strings.Join(*v, ",") }
func (v *extensionStringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value cannot be empty")
	}
	*v = append(*v, value)
	return nil
}

func extensionCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("extension requires install, approve, enable, suspend, revoke or status")
	}
	switch args[0] {
	case "install":
		return extensionInstall(args[1:], stdout, stderr)
	case "approve":
		return extensionApprove(args[1:], stdout, stderr)
	case "enable":
		return extensionTransition(args[1:], extensionv1.StateEnabled, stdout, stderr)
	case "suspend":
		return extensionTransition(args[1:], extensionv1.StateSuspended, stdout, stderr)
	case "revoke":
		return extensionTransition(args[1:], extensionv1.StateRevoked, stdout, stderr)
	case "status":
		return extensionStatus(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown extension subcommand %q", args[0])
	}
}

func extensionInstall(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("extension install", stderr)
	packagePath := flags.String("package", "", "Signed .hx package (required)")
	staging := flags.String("staging", "", "Extension staging directory (required)")
	trustedKey := flags.String("trusted-key", "", "Trusted Ed25519 public key file or base64 value (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *packagePath == "" || *staging == "" || *trustedKey == "" {
		return errors.New("extension install requires --package, --staging and --trusted-key")
	}
	signed, _, err := extensionv1.ReadArchive(*packagePath)
	if err != nil {
		return fmt.Errorf("read extension package: %w", err)
	}
	publicKey, err := loadPublicKey(*trustedKey)
	if err != nil {
		return fmt.Errorf("load trusted key: %w", err)
	}
	metadata, err := signed.Verify(map[string]ed25519.PublicKey{signed.KeyID: publicKey})
	if err != nil {
		return fmt.Errorf("verify extension package: %w", err)
	}
	store := extensionpkg.Store{Root: *staging}
	if _, err := store.Load(metadata.Manifest.ID); err == nil {
		return errors.New("extension is already installed")
	}
	destinationDir := filepath.Join(*staging, metadata.Manifest.ID)
	if err := os.MkdirAll(destinationDir, 0o700); err != nil {
		return err
	}
	destination := filepath.Join(destinationDir, metadata.Manifest.Version+".hx")
	if err := copyExclusive(*packagePath, destination); err != nil {
		return fmt.Errorf("stage extension package: %w", err)
	}
	requested := requestedCapabilities(metadata.Manifest)
	if err := store.Save(extensionpkg.Record{ID: metadata.Manifest.ID, Version: metadata.Manifest.Version, State: extensionv1.StateInstalled, Artifact: filepath.Join(metadata.Manifest.ID, metadata.Manifest.Version+".hx"), RequestedCapabilities: requested}); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "extension-installed id=%s version=%s state=%s\n", metadata.Manifest.ID, metadata.Manifest.Version, extensionv1.StateInstalled)
	return err
}

func extensionApprove(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("extension approve", stderr)
	id := flags.String("id", "", "Extension ID (required)")
	staging := flags.String("staging", "", "Extension staging directory (required)")
	allRequested := flags.Bool("all-requested", false, "Approve all manifest-requested capabilities")
	var grants extensionStringList
	flags.Var(&grants, "capability", "Capability to approve; repeatable")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *id == "" || *staging == "" || (*allRequested && len(grants) > 0) || (!*allRequested && len(grants) == 0) {
		return errors.New("approve requires --id, --staging, and either --all-requested or one or more --capability values")
	}
	store := extensionpkg.Store{Root: *staging}
	record, err := store.Load(*id)
	if err != nil {
		return fmt.Errorf("load extension: %w", err)
	}
	if record.State != extensionv1.StateInstalled {
		return fmt.Errorf("extension must be installed before approval; current state=%s", record.State)
	}
	requested := make(map[string]struct{}, len(record.RequestedCapabilities))
	for _, capability := range record.RequestedCapabilities {
		requested[capability] = struct{}{}
	}
	if *allRequested {
		grants = append(grants, record.RequestedCapabilities...)
	}
	grants = uniqueSorted(grants)
	for _, capability := range grants {
		if _, ok := requested[capability]; !ok {
			return fmt.Errorf("capability %q was not requested by the manifest", capability)
		}
	}
	record.GrantedCapabilities = grants
	if _, err := store.Transition(record, extensionv1.StateApproved); err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "extension-approved id=%s capabilities=%s state=%s\n", *id, strings.Join(grants, ","), extensionv1.StateApproved)
	return err
}

func extensionTransition(args []string, next extensionv1.PackageState, stdout, stderr io.Writer) error {
	flags := newFlagSet("extension transition", stderr)
	id := flags.String("id", "", "Extension ID (required)")
	staging := flags.String("staging", "", "Extension staging directory (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *id == "" || *staging == "" {
		return errors.New("extension transition requires --id and --staging")
	}
	store := extensionpkg.Store{Root: *staging}
	record, err := store.Load(*id)
	if err != nil {
		return fmt.Errorf("load extension: %w", err)
	}
	if next == extensionv1.StateEnabled && len(record.GrantedCapabilities) == 0 {
		return errors.New("extension cannot be enabled without approved capabilities")
	}
	updated, err := store.Transition(record, next)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "extension-state id=%s state=%s\n", updated.ID, updated.State)
	return err
}

func extensionStatus(args []string, stdout, stderr io.Writer) error {
	flags := newFlagSet("extension status", stderr)
	id := flags.String("id", "", "Extension ID (required)")
	staging := flags.String("staging", "", "Extension staging directory (required)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *id == "" || *staging == "" {
		return errors.New("extension status requires --id and --staging")
	}
	record, err := (extensionpkg.Store{Root: *staging}).Load(*id)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "extension-status id=%s version=%s state=%s capabilities=%s\n", record.ID, record.Version, record.State, strings.Join(record.GrantedCapabilities, ","))
	return err
}

func requestedCapabilities(manifest extensionv1.AdminExtensionManifest) []string {
	var capabilities []string
	for _, surface := range manifest.Surfaces {
		capabilities = append(capabilities, surface.Permissions...)
	}
	return uniqueSorted(capabilities)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func copyExclusive(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	return flags
}
