#!/usr/bin/env bash
rm beanstalk_config.zip
mkdir tmp/
cd project/
zip -r beanstalk_config.zip * .ebextensions/
mv beanstalk_config.zip ..
cd ..
rm -rf tmp/
