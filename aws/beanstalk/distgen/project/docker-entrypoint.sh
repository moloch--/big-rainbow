#!/bin/bash

python3 /opt/distgen/healthcheck.py &
python3 /opt/distgen/distgen_compute.py
