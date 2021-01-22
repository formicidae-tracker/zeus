# Season file examples
This directory explain the structyure of season file and list a series of examples.

You can do a lot of nifty things with season file. So test them using the `dieu simulate` command before starting your experiment

## List of examples

* `simple.season` : a simple day/night cycle for a single box. You can remark that if you run `dieu -c simple.season simulate -s 09:00` the first state will be 'day' and if you run `dieu -c simple.season simulate -s 22:00` the first state will be night. dieu try to infere the sarting state for you !
* `two-zones.season`: a box zone with a day/night cycle and a 'tunnel' zone with always a lot of light
* `cloudy-days.season`: a more involved example with two 'cloudy-day' with less light and humidity
* `changing-daytime.season`: an example where reccuring change in the day/night cycle slightly shifts every day by a fix amount

## Structure of a season file

Season file are based on [YAML](https://learn.getgrav.org/advanced/yaml) language.

We start by defining a list of named zone that zeus must control. Each
node declares the zone it can manage. By default this zone is called
`box`.

```yaml
zones:
  # we define a zone called 'box'
  box:
```


For each zone that have temperature and humidity monitoring (where a
Zeus board is assigned to) you can define boundary condition wich will
trigger alarms.

```yaml
zones:
  box:
    minimal-temperature: 20.0 #°C
    maximal-temperature: 31.0 #°C
    minimal-humidity: 40.0 # % R.H.
    maximal-humidity: 80.0 # % R.H.
```

Then we define all the possible states of our climate state
machine. Each states can defines desired temperature, humidity, wind,
and light (visible and UV). Each state should have a unique name. You
can define as many state you want for a zone.

```yaml
zones:
  box:
    states:
      - name: day
        temperature: 26.0 # °C
        humidity: 60.0 # % R.H.
        wind: 100 # % of max
        visible-light: 30.0 # % of max
        uv-light: 100.0 # % of max
      - name: night
        temperature: 22.0
        visible-light: 0
        uv-light: 0
```

You do not have to specify all the state value. If two state are
linked with a transition, all the misisng values will be taken from
the previous value

Finally we should define transitions from one state to another.

```yaml
zones:
  box:
    transitions:
      - from: night
        to: day
        start: 06:00
        duration: 30m
      - from: day
        to: night
        start: 17:00
        duration: 40m
```

Here we define two transitions, one from the 'night' to the 'day
state, occuring every day at 06:00 __UTC__ , and another one from
'day' to 'night' occuring every day at 17:00 __UTC__ . Indeed the use
of UTC time is mandatory to avoid changes in the expected 24h cycle if
the experiment would be run during a daylight time change in your
local timezone.

Each transition is not necersarly instantaneous, and a could use the
duration field.. Then dieu will linearly interpolate all the value
between the two steps over the desired duration. Possible suffixes are
'h' 'm' 's' and 'us'.

Furthermore transitions are not necersarly occuring everyday. Using
the `day` field, we can define a transition that will occurs only in
the experiment n days after the start of the experiment

```yaml
zones:
  box:
    transitions:
      # the following transition will occur only once after 4 day of experiment
      - from: night
        to: cloudy-day
        start: 06:00
        duration: 30m
        day: 4
      # we then go back to a normal night, we can leave it reccuring it will still occurs only once
      - from: cloudy-day
        to: night
        start: 17:00
        duration: 30m
```
