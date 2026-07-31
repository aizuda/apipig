#!/bin/bash

# 项目名称
PROJECT_NAME="apipig"

# 项目目录
PROJECT_DIR="/opt/apipig"

# 启动函数
start() {
    echo "Starting $PROJECT_NAME..."
    chmod 777 $PROJECT_DIR/$PROJECT_NAME
    cd $PROJECT_DIR
    nohup ./$PROJECT_NAME > logs/$PROJECT_NAME.out.log 2>&1 &
    echo "$PROJECT_NAME started."
}

# 停止函数
stop() {
    echo "Stopping $PROJECT_NAME..."
    PID=$(ps -ef | grep $PROJECT_NAME | grep -v grep | awk '{print $2}')
    if [ -z "$PID" ]; then
        echo "$PROJECT_NAME is not running."
    else
        kill -9 $PID
        echo "$PROJECT_NAME stopped."
    fi
}

# 检查输入参数
case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        stop
        sleep 2
        start
        ;;
    *)
        echo "Usage: $0 {start|stop|restart}"
        exit 1
        ;;
esac

exit 0
