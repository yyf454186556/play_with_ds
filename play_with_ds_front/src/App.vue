<template>
  <div id="app">
    <!-- 左侧内容容器 -->
    <div class="left-section">
      <h1>小小DND</h1>
      <textarea 
        v-model="inputText" 
        placeholder="请输入内容"
        class="custom-input"
      ></textarea>
      <button 
        @click="fetchData"
        class="borderless-btn"
      >
        DM，说句话
      </button>
      <textarea 
        v-model="responseText" 
        readonly
        class="custom-textarea"
      ></textarea>
    </div>

    <!-- 右侧固定图片容器 -->
    <img 
      :src="imageUrl" 
      alt="Dynamic Image"
      class="fixed-image"
    >
  </div>
</template>

<style scoped>
#app {
  display: flex;
  gap: 40px; /* 左右分区间距 */
  padding: 20px;
  align-items: flex-start; /* 顶部对齐 */
  max-width: 1200px;
  margin: 0 auto;
}

.left-section {
  flex: 1; /* 占据剩余空间 */
  display: flex;
  flex-direction: column;
  gap: 15px; /* 内部元素间距 */
  max-width: 600px;
}

/* 无边框按钮样式 */
.borderless-btn {
  margin-top: 40px; /* 修正了缺少单位的问题 */
  border: none;
  outline: none;
  padding: 12px 24px;
  background: #42b983;
  color: white;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.3s;
  font-size: 16px;
}

.borderless-btn:hover {
  background: #33a06f;
}

/* 输入框样式（调整为textarea） */
.custom-input {
  border: none;
  border-bottom: 2px solid #eee; /* 底部细线 */
  padding: 12px;
  font-size: 16px;
  transition: border-color 0.3s;
  resize: vertical; /* 允许垂直调整 */
  min-height: 100px; /* 设置最小高度 */
  font-family: inherit;
}

.custom-input:focus {
  border-bottom-color: #42b983;
  outline: none;
}

/* 文本域样式 */
.custom-textarea {
  border: none;
  background: #f8f8f8;
  padding: 15px;
  border-radius: 8px;
  min-height: 250px;
  resize: vertical; /* 允许垂直调整 */
  font-family: inherit;
  font-size: 14px;
  overflow: hidden;
}

/* 固定尺寸图片 */
.fixed-image {
  width: 512px;
  height: 512px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  object-fit: cover; /* 保持比例填充 */
}
</style>

<script>
import axios from 'axios';

export default {
  data() {
    return {
      inputText: '', // 输入框的内容
      responseText: '', // 文本框的内容
      imageUrl: 'e0f4e7fb-d804-42e8-9c8b-b3d69e805ea4.png', // 图片路径
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
        const response = await axios.post(`http://124.222.139.115:44444/dnd`, requestBody, {
          // headers: {
          //   'Content-Type': 'application/json' // 设置请求头
          // }
        });
        
        // 将响应数据转换为字符串并显示在文本框中
        this.responseText = JSON.stringify(response.data.success, null, 2);
        this.imageUrl = `/public/${response.data.uuid}.png`;
      } catch (error) {
        // 如果请求失败，显示错误信息
        this.responseText = `请求失败: ${error.message}`;
      }
    }
  }
};
</script>
