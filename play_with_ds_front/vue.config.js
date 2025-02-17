module.exports = {
    devServer: {
      host: '0.0.0.0', // 监听所有网络接口
      port: 44445,       // 指定端口（可自定义）
      allowedHosts: 'all', // 允许所有主机访问
      // 若遇到主机头校验问题，可尝试以下选项（根据Vue CLI版本选择）
      // disableHostCheck: true, // 旧版本选项，可能存在安全风险
      publicPath: './'
    },
    outputDir: 'dist', // 确保这个目录存在且正确
  };
