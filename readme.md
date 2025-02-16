查询并关闭占用了44444端口的进程
sudo kill -9 $(sudo lsof -t -i :44444)

路径
/repository/play_with_ds

启动
nohup go run . > output.log 2>&1 &