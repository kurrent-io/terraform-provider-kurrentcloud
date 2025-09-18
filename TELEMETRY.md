# Telemetry Implementation

This Terraform provider includes telemetry functionality to help identify usage patterns while maintaining user privacy.

## User-Agent Header

All HTTP requests to the KurrentCloud API include a User-Agent header with the following format:

```
terraform-provider-kurrentcloud/{version} Terraform/{terraform_version} Go/{go_version} ({os}/{arch})
```

Example:

```
terraform-provider-kurrentcloud/1.2.3 Terraform/1.6.0 Go/go1.23.0 (darwin/arm64)
```

## Terraform Version Detection

The Terraform version is automatically detected when the provider is configured:

- **During `terraform plan/apply`**: The actual Terraform version (e.g., "1.6.0") is detected from the Terraform CLI
- **Without Terraform context**: Falls back to "unknown" (e.g., during testing or standalone runs)

This detection happens through the Terraform Plugin SDK v2, which provides the version information when Terraform initializes the provider.

### How Version Detection Works

1. **Provider Configuration**: When Terraform runs `terraform plan` or `terraform apply`, it calls the provider's configure function
2. **SDK v2 Integration**: The `*schema.Provider` struct contains a `TerraformVersion` field populated by Terraform
3. **Automatic Detection**: The provider reads `p.TerraformVersion` and stores it for use in User-Agent headers
4. **Telemetry Integration**: All subsequent HTTP requests include the detected Terraform version

This ensures accurate version reporting for all Terraform-initiated operations while maintaining "unknown" for non-Terraform contexts.

## Build-Time Version Injection

### Manual Build

To set the provider version at build time, use the `-ldflags` flag:

```bash
go build -ldflags "-X main.version=1.2.3"
```

### Using Makefile

The project includes automated version detection in the GNUmakefile:

```bash
make build        # Builds with automatic version detection
make version      # Shows detected version information
make ci           # CI build with version injection
```

The Makefile automatically detects version using:

- `git describe --tags --always --dirty` for version
- `git rev-parse --short HEAD` for commit hash

### CI/CD Builds

For CI builds, the version will be automatically generated from git information:

- **Tagged releases**: `v1.2.3` (from git tags)
- **Development builds**: `v1.7.1-6-g400985c-dirty` (tag + commits + hash + dirty status)
- **Fallback**: `dev` if git is not available

This ensures every build has a unique, traceable version in the telemetry.

## Privacy Considerations

- Only standard User-Agent headers are sent
- No personal information, API keys, or resource names are included
- No data is sent to third-party analytics services

## Implementation Details

The telemetry is implemented through:

1. **HTTP Transport Middleware** (`client/telemetry.go`): Automatically adds User-Agent headers to all outgoing requests
2. **Version Management** (`main.go`): Sets the provider version from build-time variables
3. **Automatic Integration** (`client/client.go`): Seamlessly integrates with the existing HTTP client

The implementation preserves any existing User-Agent headers and only adds the provider information if no User-Agent is already set.
