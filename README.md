# zeus: climate control program for the FORmicidae Tracker

This program inrterface with the fort hardware to monitor and control
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

### zeus-cli

`zeus-cli` is installed by using snap.

``` bash
sudo snap install fort-zeus-cli
sudo snap alias fort-zeus-cli zeus-cli
```

You should be able to scan node over the network with

``` bash
zeus-cli scan
```

you may run into a tcp lookup error. This is due to a limitation of
snap regarding `.local` network addresses. It can be solved using the
following commands once.

``` bash
sudo apt install nscd
sudo service snapd restart
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
