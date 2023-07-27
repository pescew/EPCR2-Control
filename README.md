# EPCR2-Control
CLI utility to control legacy Digital Loggers Ethernet Power Controller II (EPCR2)

Tested on the lastet firmware version (v1.9.1 Sep 02 2019).

Set environment variables or edit config.yaml with correct settings for your environment.

```
flags:
  -action string
        action to perform (on, off, cycle) (default "on")
  -outlet int
        outlet number to control. 0 for all outlets
```
