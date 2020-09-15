This is the docker-compose setup.



# FAQ

## How do I start the development environment?

You run `docker-compose up` from the `docker/` directory. It will build any services defined in the `docker-compose.yml` file and start them in the (hopefully) right order.

```shell
docker-compose up
```

Add the `-d` option to run the services in the background; you can then use `docker-compose logs` to check container output.

## How do I connect to services inside the network environment?

Inside the created network, the services can connect to each other by hostname. On the host itself you can not resolve the internal hostnames; you need to connect to specific services by IP address.

You can list IP addresses for each running container with the following command:

```shell
docker inspect -f '{{.Name}} {{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker-compose ps -aq)
```

To also list service ports, use:

```shell
docker inspect --format='{{.Name}} {{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}{{range $p, $conf := .NetworkSettings.Ports}} {{$p}} {{end}}' $(docker-compose ps -aq)
```

You can also list containers by inspecting a given network by name:

```shell
docker network inspect -f '{{range .Containers}}{{println .Name .IPv4Address}}{{end}}' urnnet
```

All of these commands will return a list of IPs somewhere around the internal 172.17.0.0/16 range, which you can connect to from the host machine.


## How do I remove the database and initialise the SQL schema?

The `postgres` image only executes the database initialisation scripts on first run. When it finds a pre-existing database, it will refuse to run the database initialisation scripts. To start all over, remove the database service and restart the `postgres` container:

```shell
docker-compose rm db
docker-compose start db
```

