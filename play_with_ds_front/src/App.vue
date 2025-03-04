<template>
  <div>
    <div id="app">
      <!-- 左侧内容容器 -->
      <div class="left-section">
        <h1>小小DND</h1>
        <input v-model="inputName" placeholder="你的id,用于标识此次冒险。只能是数字" class="custom-input-name"/>
        <textarea v-model="inputText" placeholder="第一次会话时，请输入您的角色描述。包含时间，场景，任务描述等等。后续会依据第一次描述生成图片~" class="custom-input"></textarea>
        <button  @click="fetchData"class="borderless-btn">DM，说句话</button>
        <textarea v-model="responseText" readonly class="custom-textarea"></textarea>
      </div>
      <!-- 右侧固定图片容器 -->
      <div class="fied-image-div">
        <img :src="currentImage" alt="Dynamic Image"class="fixed-image">
        <div>
          <button @click="showPreview" class="borderless-btn-arrorw"> < </button>
          <button @click="showNext" class="borderless-btn-arrorw"> > </button>
        </div>
      </div>
    </div>
    <!-- 对话记录显示区域 -->
    <div class="history-box">
      <div v-for="(item, index) in history" :key="index" class="message" :class="{'user-message': item.type === 'user', 'dm-message': item.type !== 'user'}">
        {{ item.type === 'user' ? '玩家' : 'DM' }}: {{ item.content }}
      </div>
    </div>
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

.history-box {
  margin-top: 20px;
  margin-right: 20px;
  margin-left: 20px;
  border: 1px solid #ccc;
  padding: 10px;
  height: 300px;
  padding: 40px;
  max-width: 1200px;
  overflow-y: auto;
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
  margin-top: 10px; /* 修正了缺少单位的问题 */
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

/* 无边框按钮样式 */
.borderless-btn-arrorw {
  margin-right: 20px;
  border: none;
  outline: none;
  background: #42b983;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.3s;
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
  min-height: 130px; /* 设置最小高度 */
  font-family: inherit;
  resize: none;
}

.custom-input-name {
  border: none;
  border-bottom: 2px solid #eee; /* 底部细线 */
  padding: 12px;
  font-size: 16px;
  transition: border-color 0.3s;
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
  height: 205px;
  resize: vertical; /* 允许垂直调整 */
  font-family: inherit;
  font-size: 14px;
  overflow: hidden;
  resize: none;
}

/* 固定尺寸图片 */
.fixed-image {
  width: 512px;
  height: 512px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  object-fit: cover; /* 保持比例填充 */
}

.fixed-image-div {
  flex: 1; /* 占据剩余空间 */
  display: flex;
  flex-direction: column;
  width: 512px;
  height: 512px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
  object-fit: cover; /* 保持比例填充 */
}

/* 玩家消息的样式 */
.user-message {
  color: black; /* 玩家消息黑色 */
}

/* DM消息的样式 */
.dm-message {
  color: blue; /* DM消息蓝色 */
}
</style>

<script>
import axios from 'axios';

export default {
  data() {
    return {
      inputText: '', // 输入框的内容
      inputName: '', // 用户id
      responseText: '', // 文本框的内容
      images: ['welcome.png'], // 图片路径
      currentImageIndex: 0, // 当前显示的图片索引
      history: [],
    };
  },
  computed: {
    currentImage() {
      return this.images[this.currentImageIndex] || '';
    }
  },
  methods: {
    async fetchData() {
      try {
        const requestBody = {
          auth: "zzyztyy",
          story_id: this.inputName,
          content: this.inputText
        }

        // 添加用户消息到历史记录
        this.history.push({
            type: 'user',
            content: this.inputText
        });

        // 发送 HTTP GET 请求，将输入框的内容作为参数
        const response = await axios.post(`http://124.222.139.115:44444/dnd`, requestBody, {
          // headers: {
          //   'Content-Type': 'application/json' // 设置请求头
          // }
        });        
        // 将响应数据转换为字符串并显示在文本框中
        this.responseText = JSON.stringify(response.data.success, null, 2);
        if (response.data.uuid) {
           //this.imageUrl = `/public/${response.data.uuid}.png`;
           this.addImage(`/public/${response.data.uuid}.png`);
           console.log("hello world")
        }
        // 添加系统回复
        this.history.push({
            type: 'system',
            content: this.responseText
        });
      } catch (error) {
        // 如果请求失败，显示错误信息
        this.responseText = `请求失败: ${error.message}`;
      }
    },
    showPreview() {
      if (this.currentImageIndex > 0) {
        this.currentImageIndex--;
      }
    },
    showNext() {
      if (this.currentImageIndex < this.images.length) {
        this.currentImageIndex++;
      }
    },
    addImage(imageUrl) {
      this.images.push(imageUrl);
      this.currentImageIndex = this.images.length - 1;
    }
  }
};
</script>

