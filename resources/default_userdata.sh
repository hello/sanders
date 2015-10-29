#!/bin/sh

# When updated, this file should replace the file at s3://hello-deploy/userdata/default_userdata.sh

export AWS_DEFAULT_REGION={default_region}
PKG_NAME={app_name}_{app_version}_amd64.deb
sudo update-java-alternatives -s /usr/lib/jvm/java-1.{java_version}.0-openjdk-amd64
aws s3 cp s3://hello-deploy/packages/com/hello/suripu/{app_name}/{app_version}/$PKG_NAME /tmp/
sudo dpkg -i /tmp/$PKG_NAME
aws s3 cp s3://hello-deploy/pkg/kenko/kenko_latest_amd64.deb /tmp/
sudo dpkg -i /tmp/kenko_latest_amd64.deb
aws s3 cp s3://hello-deploy/pkg/papertrail/papertrail_1.1_amd64.deb /tmp/
sudo dpkg -i /tmp/papertrail_1.1_amd64.deb
