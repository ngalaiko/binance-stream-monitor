# Binance stream monitor

## Configuration

1. Command line arguments

| Command line            | Default                  | Description                                             |
| ----------------------- | ------------------------ | ------------------------------------------------------- |
| verbose                 | false                    | Enable verbose logging                                  |
| alert-on                | []                       | Symbol and price to alert on, for example BTCUSDT>51000 |

## Running

1. Locally
```
$ cd src && \
    go run ./cmd/monitor/main.go \
        --alert-on "BTCUSDT>51000" \
        --alert-on "ADABTC>0"
```

2. Docker
```
$ docker build . -t monitor
$ docker run -it monitor \
    --alert-on "BTCUSDT>51000" \
    --alert-on "ADABTC>0"
```
