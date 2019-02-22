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
