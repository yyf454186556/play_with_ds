<template>
  <div id="app">
    <h1>Vue.js HTTP 请求示例</h1>
    <input v-model="inputText" placeholder="请输入内容" />
    <button @click="fetchData">发送请求</button>
    <textarea v-model="responseText" readonly></textarea>
  </div>
</template>

<script>
import axios from 'axios';

export default {
  data() {
    return {
      inputText: '', // 输入框的内容
      responseText: '' // 文本框的内容
    };
  },
  methods: {
    async fetchData() {
      try {
        const requestBody = {
          auth: "zzyztyy",
          name: "yyf",
          content: this.inputText
        }


        // 发送 HTTP GET 请求，将输入框的内容作为参数
        const response = await axios.post(`http://118.89.203.67:44444/ask20`, requestBody, {
          // headers: {
          //   'Content-Type': 'application/json' // 设置请求头
          // }
        });
        
        // 将响应数据转换为字符串并显示在文本框中
        this.responseText = JSON.stringify(response.data, null, 2);
      } catch (error) {
        // 如果请求失败，显示错误信息
        this.responseText = `请求失败: ${error.message}`;
      }
    }
  }
};
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  text-align: center;
  margin-top: 60px;
}

input, button, textarea {
  margin: 10px;
  padding: 10px;
  font-size: 16px;
}

textarea {
  width: 300px;
  height: 150px;
  resize: none;
}
</style>