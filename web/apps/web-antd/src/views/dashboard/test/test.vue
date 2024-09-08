<script setup lang="ts">
import { useRefresh } from '@vben/hooks';

const { refresh } = useRefresh();
</script>

<template>
  <div>
    <p>我是测试页面111</p>
    <button type="submit" @click="refresh">点我测试</button>
  </div>
</template>

<style scoped></style>
