import type { Ref } from 'vue';
import { ref } from 'vue';

export function useLoading(externalLoading?: Ref<boolean>) {
    // 如果外部传入了 loading，就用外部的，否则创建一个新的
    const loading = externalLoading || ref(false);

    async function withLoading<T>(asyncFn: () => Promise<T>): Promise<T> {
        try {
            loading.value = true;
            const resulthooks = await asyncFn();
            return resulthooks;
        } finally {
            loading.value = false;
        }
    }

    return { loading, withLoading };
}
