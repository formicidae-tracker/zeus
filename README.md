# dieu: climate control program for the FORmicidae Tracker

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

## Getting started

### Prerequesite

If actual climate should be controlled the host should run Linux and
have installed `slcan-utils` packages.

Otherwise if just simulation is required, you only need to install go
version 1.11. You can follow installation instructions
[here](https://golang.org/doc/install).

Please do not forget to setup your GOPATH and [test your
installation](https://golang.org/doc/install#testing)

### Installing

``` bash
go get -u git.tuleu.science/fort/dieu/dieu
go install git.tuleu.science/fort/dieu/dieu
```

You should now be able to run the `dieu` command.


## Testing a season file

You can find sample season file in `examples/` subfolder. You can test
the climate it will produce using `dieu simulate` subcommand.

``` bash
cd examples
dieu -c simple.season simulate [-d 10] [-s 06:00]
```

Will list on stdout the states dieu will go through over a duration of
7 days starting from the current time. You can change this duration
using the `-d` flags and defines the starting time of the simulation
with the `-s` flags.


You can find more information in
[examples](/fort/dieu/src/master/examples/list.md).


## Authors

  * Alexandre Tuleu - Initial Work

## License

This project is licensed under the GPL version 3 license
