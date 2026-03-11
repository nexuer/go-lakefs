import { ofetch } from 'ofetch'



export const api = ofetch.create({
    baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',

    onRequestError({ error }) {
        window.$dialog?.error({
            title: "请求接口错误",
            content: error.message,
            positiveText: '确定',
        })
    },
    onResponseError({ response }) {
        const data = JSON.parse(response._data)
        window.$dialog?.error({
            title: "请求接口错误",
            content: data.message,
            positiveText: '确定',
        })
    }
})