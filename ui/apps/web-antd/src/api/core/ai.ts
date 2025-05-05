import { requestClient } from '#/api/request';

export async function chat(data: any) {
    return requestClient.post('/ai/chat', data);
}