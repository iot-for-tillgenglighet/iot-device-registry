# iot-device-registry
A service that manages information and status for all IoT devices known to this hub

## End to end testing

Start the service in a composed environment with database and message queue, and then invoke the runner script for starting a suite of tests against the service using Robot Framework.

```
docker-compose -f deployments/docker-compose.yml build
docker-compose -f deployments/docker-compose.yml up
python3 test/run_robot_tests.py
```

If you do not have the robotframework and robotframework-requests libraries installed, the runner script will instruct you to install them using pip3.
