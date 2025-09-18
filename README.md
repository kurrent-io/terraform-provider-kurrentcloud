# Terraform Provider for Kurrent Cloud

This repository contains a [Terraform][terraform] provider for provisioning resources in [Kurrent Cloud][esc] (formerly Event Store Cloud).

## Documentation

You can browse documentation on the [Terraform provider registry](https://registry.terraform.io/providers/kurrent-io/kurrentcloud/latest/docs).

## Migration from EventStore Cloud Provider

If you're migrating from the legacy `EventStore/eventstorecloud` provider to `kurrent-io/kurrentcloud`, please see our comprehensive **[Migration Guide](MIGRATION.md)**.

### Quick Migration Summary

1. **Provider Registry**: Change from `EventStore/eventstorecloud` to `kurrent-io/kurrentcloud`
2. **Resource Prefixes**: `eventstorecloud_*` resources are deprecated, use `kurrentcloud_*` instead
3. **Version 2.0.0+**: Both prefixes supported for backward compatibility

For detailed instructions, troubleshooting, and examples, see **[MIGRATION.md](MIGRATION.md)**.

## Contributing

The Kurrent Cloud Terraform provider is released under the Mozilla Public License version 2, like most Terraform
providers. We welcome pull requests and issues! We adhere to the [Contributor Covenant][codeofconduct] Code of Conduct,
and ask that contributors familiarize themselves with it. We also have a set of [Contributing Guidelines][contributing].

[terraform]: (https://terraform.io)
[esc]: https://kurrent.io/kurrent-cloud/
[codeofconduct]: https://github.com/kurrent-io/terraform-provider-kurrentcloud/tree/trunk/CODE-OF-CONDUCT.md
[contributing]: https://github.com/kurrent-io/terraform-provider-kurrentcloud/tree/trunk/CONTRIBUTING.md
