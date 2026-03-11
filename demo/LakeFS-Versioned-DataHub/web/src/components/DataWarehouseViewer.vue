<template>
  <n-flex>
    <div style="width: 450px">
      <n-input 
        v-model:value="listReposRequest.search" 
        placeholder="搜索仓库..." 
        @input="listRepos" 
      />
    </div>
    <n-button type="primary" @click="withLoading(handleCreateRepo)">创建仓库</n-button>
  </n-flex>
  <n-data-table
    style="margin: 16px 0 0 0"
    :columns="columns"
    :data="data"
    :bordered="false">

  </n-data-table>


  <n-drawer :width="502" v-model:show="filesDrawerVisible">
    <n-drawer-content closable>
      <template #header>
        上传文件 | {{ currentRepo?.id }} 
      </template>
      <n-form>
        <n-form-item :show-label="false">
          <n-upload 
            multiple 
            directory-dn
            :max="10"
            :custom-request="handleUploadRequest"
          >
            <n-upload-dragger>
              <div style="margin-bottom: 12px">
                <n-icon size="48" :depth="3">
                  <ArchiveIcon />
                </n-icon>
              </div>
              <n-text style="font-size: 16px">
                点击或者拖动文件到该区域来上传
              </n-text>
              <n-p depth="3" style="margin: 8px 0 0 0">
                最多上传10个文件
              </n-p>
            </n-upload-dragger>
          </n-upload>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space :size="16">
          <n-button @click="() => {
            filesDrawerVisible = false
          }">
            取消
          </n-button>
          <n-button type="primary" :loading="loading" @click="withLoading(submitAddFiles)">
            确认
          </n-button>
        </n-space>
       </template>
    </n-drawer-content>
  </n-drawer>

  <n-modal v-model:show="filePreviewModalVisible">
    <n-card 
      style="width: 70%" 
      :bordered="false"
    >
      <template #header>
        {{ currentRepo?.id}} / main / {{ currentObject?.path }}
      </template>
      <n-alert type="info" size="small">
        只能查看50KB的内容
      </n-alert>
      <pre class="streaming-pre" style="margin-top: 6px;">{{ displayedText }}</pre>
    </n-card>
  </n-modal>

  <n-drawer width="70%" v-model:show="filesTableDrawerVisible">
    <n-drawer-content closable>
      <template #header>
        文件列表 | {{ currentRepo?.id }} 
      </template>
      <n-data-table
        :columns="filesTableColumns"
        :data="currentFiles"
        :bordered="false">
      </n-data-table>
    </n-drawer-content>
  </n-drawer>

  <n-drawer :width="502" v-model:show="drawerVisible" title="创建仓库">
    <n-drawer-content title="创建仓库" closable>
      <n-form :model="createRepoRequest">
        <n-form-item label="仓库名字">
          <n-input v-model:value="createRepoRequest.name" placeholder="请输入名字"></n-input>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space :size="16">
          <n-button @click="() => {
            drawerVisible = false
          }">
            取消
          </n-button>
          <n-button type="primary" :loading="loading" @click="withLoading(submitCreateRepo)">
            确认
          </n-button>
        </n-space>
       </template>
    </n-drawer-content>
  </n-drawer>

</template>
<script setup lang="tsx">
import { NButton, NFlex, NPopconfirm, type DataTableColumns, type FormRules, type UploadCustomRequestOptions } from 'naive-ui'
import { ArchiveOutline as ArchiveIcon } from '@vicons/ionicons5'
import { useLoading } from "@/hooks/loading";
import { onMounted, reactive, ref } from 'vue';
import type { Repo, ListReposRequest, CreateRepoRequest, CreateFilesRequest, Object } from "@/api"
import { fetchCreateFiles, fetchCreateRepo, fetchDeleteObject, fetchDeleteRepo, fetchDownloadFile, fetchFilePresignUrl, fetchFilePreview, fetchListObjects, fetchListRepos } from "@/api"
import dayjs from 'dayjs';

const data = ref<Repo[]>([])

const loading = ref(false)

const displayedText = ref("")

const listReposRequest = reactive<ListReposRequest>({
  search: ""
})

function newCreateRepoRequest(): CreateRepoRequest {
  return {
    name: ""
  }
}

function newCreateFilesRequest(): CreateFilesRequest {
  return {
    files: []
  }
}

const createRepoRequest = ref<CreateRepoRequest>(newCreateRepoRequest())
const createFilesRequest = ref<CreateFilesRequest>(newCreateFilesRequest())

const { withLoading } = useLoading(loading);

const drawerVisible = ref(false)
const filesDrawerVisible = ref(false)
const filesTableDrawerVisible = ref(false)
const filePreviewModalVisible = ref(false)
const currentRepo = ref<Repo>()


onMounted(async () => {
  await listRepos()
})

async function handleUploadRequest({ file, onFinish, onError, onProgress }: UploadCustomRequestOptions) {
  console.log("---", file)
  try {
    // 1. 为当前文件请求预签名 URL
    const resp = await fetchFilePresignUrl(currentRepo.value?.id, file.name)
    if (!resp.presigned_url) {
      throw new Error("获取签名地址失败")
    }
    // 2. 直接 PUT 到 S3/MinIO 预签名 URL
    const xhr = new XMLHttpRequest()
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) {
        onProgress({ percent: Math.round((e.loaded / e.total) * 100) })
      }
    }


    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        const checksum = xhr.getResponseHeader("etag")
        onFinish()  // 更新 n-upload 的文件状态
        // fileList.value.push(fileInfo)  // 收集到列表，稍后提交
        createFilesRequest.value?.files.push({
          staging: {
            physical_address: resp.physical_address,
            presigned_url: resp.presigned_url,
            presigned_url_expiry: resp.presigned_url_expiry,
          },
          user_metadata: {
            filename: file.name,
          },
          size_bytes: file.file?.size,
          content_type: file.file?.type,
          checksum: checksum,
          mtime: Math.floor(file.file?.lastModified / 1000),
        })
      } else {
        onError()
      }
    }

    xhr.onerror = () => onError()
    xhr.open('PUT', resp.presigned_url, true)
    // 重要：设置 Content-Type，否则 MinIO/S3 可能拒绝或签名不匹配
    xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream')
    xhr.send(file.file)
  } catch(err) {
    window.$message?.error(`上传${file.name}失败：${err}`)
    onError()
  }
}

async function listRepos() {
  const resp = await fetchListRepos(listReposRequest)
  data.value = resp
}

function formatTs(ts: number): string {
  return dayjs(ts * 1000).format("YYYY-MM-DD HH:mm:ss")
}

const filesTableColumns: DataTableColumns<Object> = [
  {
    title: "文件",
    key: "path",
    sorter: 'default'
  },
  {
    title: "文件大小",
    key: "size_bytes",
    sorter: 'default'
  },
  {
    title: "最近修改时间",
    key: "mtime",
    sorter: (row1: Object, row2: Object) => row1.mtime - row2.mtime,
    render(row: Object) {
      return formatTs(row.mtime)
    }
  },
  {
    title: "文件类型",
    key: "content_type",
    sorter: 'default'
  },
  {
    title: "操作",
    key: "operate",
    width: 200,
    render: row => (
      <NFlex>
        <NButton 
          text 
          ghost 
          size="small" 
          type="primary"
          onClick={ () => hadnleFilePreview(row) }
          >
          查看
        </NButton>
        <NButton 
          text 
          ghost 
          size="small" 
          type="primary"
          onClick={ () => submitDownloadFile(row) }
          >
          下载
        </NButton>
        <NPopconfirm
            onPositiveClick={() => submitDeleteFile(row)}
            positiveButtonProps={{
              loading: loading.value,
            }}
          >
            {{
              default: () => "确认删除吗",
              trigger: () => (
                <NButton type="error" text ghost size="small">
                  删除
                </NButton>
              ),
            }}
          </NPopconfirm>
      </NFlex>
    )
  }
]

const columns: DataTableColumns<Repo> = [
  {
    title: "仓库",
    key: "id",
    sorter: 'default'
  },
  {
    title: "创建时间",
    key: "creation_date",
    sorter: (row1: Repo, row2: Repo) => row1.creation_date - row2.creation_date,
    render(row: Repo) {
      return formatTs(row.creation_date)
    }
  },
  {
    title: "存储命名空间",
    key: "storage_namespace",
  },
  {
    title: "操作",
    key: "operate",
    width: 200,
    render: row => (
      <NFlex>
        <NButton 
          text 
          ghost 
          size="small" 
          type="primary"
          onClick={ () => handleFilesTable(row) }
          >
          文件列表
        </NButton>
        <NButton 
          text 
          ghost 
          size="small" 
          type="primary"
          onClick={ () => handleAddFiles(row) }
          >
            上传文件
        </NButton>
        <NPopconfirm
            onPositiveClick={() => submitDelete(row)}
            positiveButtonProps={{
              loading: loading.value,
            }}
          >
            {{
              default: () => "确认删除吗",
              trigger: () => (
                <NButton type="error" text ghost size="small">
                  删除
                </NButton>
              ),
            }}
          </NPopconfirm>
      </NFlex>
    )
  }
]

const currentFiles = ref<Object[]>([])

async function handleFilesTable(row: Repo) {
  filesTableDrawerVisible.value = true
  currentRepo.value = row
  await listFiles()
}

async function listFiles() {
  currentFiles.value = []
  const resp = await fetchListObjects(currentRepo.value?.id)
  if (resp.length > 0) {
    currentFiles.value = resp
  }
}

const currentObject = ref<Object>()



function handleAddFiles(row: Repo) {
  filesDrawerVisible.value = true
  createFilesRequest.value = newCreateFilesRequest()
  currentRepo.value = row
}

async function submitAddFiles() {
  console.log("++++", createFilesRequest.value)
  const resp = fetchCreateFiles(currentRepo.value?.id, createFilesRequest.value)
  resp.then(() => {
    listRepos()
    filesDrawerVisible.value = false
  })
}

async function handleCreateRepo() {
  drawerVisible.value = true
  createRepoRequest.value = newCreateRepoRequest() 
}

async function submitCreateRepo() {
  const resp = fetchCreateRepo(createRepoRequest.value)
  resp.then(() => {
    listRepos()
    drawerVisible.value = false
  })
}

async function submitDelete(row: Repo) {
  const resp = fetchDeleteRepo(row.id)
  resp.then(() => {
    listRepos()
  })
}

async function submitDeleteFile(row: Object) {
  const resp = fetchDeleteObject(currentRepo.value?.id, row.path)
  resp.then(() => {
    listFiles()
  })
}

async function hadnleFilePreview(row: Object) {
  displayedText.value = ""
  filePreviewModalVisible.value = true
  currentObject.value = row

  const resp = await fetchFilePreview(currentRepo.value?.id, row.path)

  const reader = resp.getReader();
  const decoder = new TextDecoder('utf-8'); // 负责将二进制流转为文本

  while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      // 关键点：将收到的二进制块 (Uint8Array) 转为字符串并追加
      const chunk = decoder.decode(value, { stream: true });
      displayedText.value += chunk; 
      
    }
}

async function submitDownloadFile(row: Object) {
  const resp = await fetchDownloadFile(currentRepo.value?.id, row.path)
  console.log(resp)
  // 1. 创建 Blob URL
  const url = window.URL.createObjectURL(resp);
  
  // 2. 创建隐藏的 a 标签模拟点击下载
  const link = document.createElement('a');
  link.href = url;
  link.download = `${currentRepo.value?.id}-main-${row.path}`; // 自定义文件名
  document.body.appendChild(link);
  link.click();
  
  // 3. 释放内存
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

</script>

<style scoped>
.streaming-pre {
  background: #1e1e1e;
  color: #dcdcdc;
  padding: 15px;
  border-radius: 8px;
  white-space: pre-wrap;       /* 自动换行 */
  word-wrap: break-word;       /* 防止长字符串撑破布局 */
  max-height: 80vh;
  overflow-y: auto;
  font-family: monospace;
}
</style>