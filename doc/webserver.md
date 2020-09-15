# web server

## configuration

This application tries to follow the [twelve-factor app](https://12factor.net/) guidelines and retrieves its configuration from the environment.

Container environments use a `.env` file, and also [`systemd` unit files](https://www.freedesktop.org/software/systemd/man/systemd.exec.html#Environment) can use variables and `.env` files with the keywords `Environment` and `EnvironmentFile` respectively.

It's especially important to make sure the [default Postgresql environments variables](https://www.postgresql.org/docs/current/libpq-envars.html) are set correctly so services can connect to the database.

## logging and output

The service writes its log stream to `STDOUT`. It is up to the environment to decide what to do with this output, to redirect it to a log aggregation service or write to a file.

Logging output is quite minimal: depending on debugging configuration a few lines per request at most.

If the program exists abnormally, an error will be written to `STDERR`; see also next section.

## exit codes

In case the program exits with an error, an error message will be written to `STDERR`.

If the program encounters an error during startup – most likely due to invalid arguments or configuration errors – the exit code will be `1`.
If this happens, it would be a good idea to check the provided parameters.

If on the other hand the error were to occur during runtime – after the webserver has started serving requests – the exit code will be `2`.
These errors ought to be very rare and should be considered critical bugs to be fixed.

In principle, services should not exit by themselves after starting up successfully.

## signals

The server listens for signals `SIGINT` and `SIGTERM`. When a signal is received, the program will attempt a graceful shutdown: ongoing requests are handled but new requests are rejected. If after a grace period there are still lingering connections, the server will forcefully shutdown.

Signal `SIGINT` most likely originates from terminal interrupts, i.e. `ctrl-c`.

Signal `SIGTERM` is sent by the OS on system or service shutdown, for instance by init, systemd, docker, kubernetes or really anything POSIX-compatible.
