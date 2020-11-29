# simulator

## Configuration
Sample configuration files are in ```config/simulator```. Each configuration file contains some general settings and specifies the topoloy to simulate. 

## Setup
Prior to running the simulator for the first time, run 
```
    sudo sysctl -w net.ipv6.conf.default.accept_ra=0
    sudo sysctl -w net.ipv4.ip_forward=1
```

Make sure you have Mahimahi installed and are able to run it. Try running ```mm-delay 1``` to test if you haven't used it before. 

## Running
```
    go build
    sudo ./simulator -config=[config path] -time=[seconds to run for]
```
