查询并关闭占用了44444端口的进程
sudo kill -9 $(sudo lsof -t -i:44444)

路径
/repository/play_with_ds

启动
nohup go run . > output.log 2>&1 &

前端
cd play_with_ds_front

npm install
npm run build
systemctl restart nginx