<script setup lang="ts">
import { createTextVNode, defineComponent } from 'vue';
import DataWarehouseViewer from './components/DataWarehouseViewer.vue'
import { useDialog, useNotification } from 'naive-ui';


// content
const ContextHolder = defineComponent({
  name: 'ContextHolder',
  setup() {
    function register() {
      window.$notification = useNotification();
      window.$dialog = useDialog();
    }

    register();

    return () => createTextVNode();
  }
});

</script>

<template>

  <n-config-provider>
    <n-dialog-provider>
      <n-notification-provider>
        <ContextHolder />
        <DataWarehouseViewer />
      </n-notification-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>
