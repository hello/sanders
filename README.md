

## Sanders for deployment

Workflow after AMI has been built for the version you want to deploy:

1. `sanders deploy` deploys **ONE** instance with the version specified.
2. `sanders confirm` once we have verified that the new application is working well, it will deploy **N** instances.
3. `sanders sunset` to sunset the previous version. Only sunset when all new instances are up and running.



To deploy the app to our `canary` environment, run the command `sanders canary`. It will kill the current instance and spin up the new version.

**Important**: `sanders canary` is **NOT** HA, there will be downtime between killing the old instance and spinning up a new one.

## Sanders (jabil branch) for Jabil

TODO