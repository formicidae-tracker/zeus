# FORmicidae Tracker (FORT) : Environmental Control Software
[![DOI](https://zenodo.org/badge/177280543.svg)](https://zenodo.org/doi/10.5281/zenodo.10019136)

The [FORmicidae Tracker (FORT)](https://formicidae-tracker.github.io) is an advanced online tracking system designed specifically for studying social insects, particularly ants and bees, FORT utilizes fiducial markers for extended individual tracking. Its key features include real-time online tracking and a modular design architecture that supports distributed processing. The project's current repositories encompass comprehensive hardware blueprints, technical documentation, and associated firmware and software for online tracking and offline data analysis.

The `zeus` program interface with the fort hardware to monitor and control
the climate for the tracking chamber. It features:

* Monitoring and logging of all climate data
  * Raising alarms on exceptional conditions:
  * Temperature/Humidity outside of user-defined limits
  * When the system detects he is not capable to humidify/heat/cool
    the chamber enough to match the desired climatic conditions
  * When hardware issue may prevent the system to reach its target
    (water tank empty, faulty fan, electronic device disconnected)
* YAML based configuration file for setting up the climate state
  machine


The program is spit in two, `zeus` that is meant to be runned as a
service and the `zeus-cli` program to start/stop climate on nodes and
simulate season files.

`zeus` is not meant to be used directly by user of the FORmicidae
Tracker. It should be installed and configured properly by site
administrators (with certainly the help of the [FORT ansible
configuration](https://github.com/formicidae-tracker/fort-configuration)

## Getting started

### Removes legacy snap installation

As ov v0.4.0, snap should not be used anymore and communication will
be incompatible with older versions.

``` bash
sudo snap remove fort-zeus-cli
```

#### Install `zeus-cli` with go tool

```bash
go install github.com/formicidae-tracker/zeus/cmd/zeus-cli@latest
```

If your tracking environment requires a specifig version, replace
`@latest` with the correct version name, for example `@v0.4.0` for
version `v0.4.0`.

Note: if you do not have go installed on your system, simply install
it from https://golang.org.

### Add local completion handler (optional)

If you use bash:

```bash
mkdir -p ~/.local/share/bash-completion/completions
cat <<EOF > ~/.local/share/bash-completion/completions/zeus-cli.bash
_zeus_cli_completion() {
    # All arguments except the first one
    args=("${COMP_WORDS[@]:1:$COMP_CWORD}")

    # Only split on newlines
    local IFS=$'\n'

    # Call completion (note that the first element of COMP_WORDS is
    # the executable itself)
    COMPREPLY=($(GO_FLAGS_COMPLETION=1 zeus-cli "${args[@]}"))
    return 0
}

complete -F _zeus_cli_completion zeus-cli
EOF
```

If you use fish:

```fish
mkdir -p ~/.config/fish/completions
echo complete -c zeus-cli -f -a \""(GO_FLAGS_COMPLETION=1 zeus-cli (string split ' ' (commandline -cp)))"\" > ~/.config/fish/completions/zeus-cli.fish
```

## Testing a season file

You can find sample season file in `examples/` subfolder. You can test
the climate it will produce using `zeus-cli simulate <file>` subcommand.

``` bash
cd examples
zeus-cli simulate [-d 10] [-s 06:00] simple.season
```

Will list on stdout the states dieu will go through over a duration of
7 days starting from the current time. You can change this duration
using the `-d` flags and defines the starting time of the simulation
with the `-s` flags.


You can find more information in
[examples](/examples/list.md).

### Starting/stopping climate on a node

Climate could be started on a node with the command

``` bash
zeus-cli start <node> <file>
```

The snap install tab auto-completion for your shell that will discover
available node on the local network and complete them.

Climate can be stopped using the command

``` bash
zeus-cli stop <node>
```

### `zeus`

It is highly advised to use the ansible configuration repository:
https://github.com/formicidae-tracker/fort-configuration/


## Authors

  * Alexandre Tuleu - Initial Work

## License

This project is licensed under the LGPL version 3 license
