# Developing `terraform-provider-kurrentcloud`

Starting from Terraform v0.14 there is an [ability to override provider path](https://www.terraform.io/docs/cli/config/config-file.html#development-overrides-for-provider-developers) in development purposes:

```
provider_installation {
  dev_overrides {
    "kurrent-io/kurrentcloud" = "path/to/go/bin"
  }
  direct {}
}
```

This configuration can be saved to `~/.terraformrc` to have a global effect or to custom path. `TF_CLI_CONFIG_FILE` environment variable should be set according to the path to file.

After that it's just enough to build provider binary to use it

```
make install
```

Documentation is generated using `go generate` to run the [tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) cli tool, which allows us to use the provider code itself as the source for documentation on its various fields and properties.

Most resources can be generated automatically by the plugin, but for some it's better to control the overall template so we can write some parts manually. In these cases the hand-written portion of the docs is found in the templates contained in the [`templates`](./templates) directory.

You can check that this is working by running:

```
make generate
```

To confirm that all of the tests gating CI pass, run

```
make ci
```
