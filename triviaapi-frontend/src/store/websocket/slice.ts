import { createSlice, PayloadAction } from '@reduxjs/toolkit';
import { WebSocketState } from './types';
import { WebSocketMessage } from '../../types/websocket';

// Начальное состояние
const initialState: WebSocketState = {
  isConnected: false,
  isConnecting: false,
  error: null,
  lastMessage: null,
  reconnectAttempts: 0,
  lastConnectedAt: null,
  lastDisconnectedAt: null,
  messageQueue: [],
  shardId: null
};

const websocketSlice = createSlice({
  name: 'websocket',
  initialState,
  reducers: {
    // Экшен при начале подключения
    wsConnecting: (state) => {
      state.isConnecting = true;
      state.isConnected = false;
      state.error = null;
    },
    // Экшен при успешном подключении
    wsConnected: (state) => {
      state.isConnected = true;
      state.isConnecting = false;
      state.error = null;
    },
    // Экшен при закрытии соединения
    wsDisconnected: (state, action: PayloadAction<{ code: number; reason: string }>) => {
      state.isConnected = false;
      state.isConnecting = false;
      // Записываем ошибку, только если закрытие не было штатным (код 1000)
      // и если не было ошибки аутентификации (код 4001), т.к. она обрабатывается отдельно
      if (action.payload.code !== 1000 && action.payload.code !== 4001) {
        state.error = `WebSocket closed: ${action.payload.code} ${action.payload.reason || 'No reason specified'}`;
      } else if (action.payload.code === 1000) {
        state.error = null; // Очищаем ошибку при штатном закрытии
      }
    },
    // Экшен при ошибке WebSocket
    wsError: (state, action: PayloadAction<{ error: string }>) => {
      state.error = action.payload.error;
    },
    // Экшен при получении нового сообщения
    wsMessage: (state, action: PayloadAction<WebSocketMessage>) => {
      state.lastMessage = action.payload;
    },
    // Экшен для очистки ошибки
    clearWSError: (state) => {
      state.error = null;
    },
  },
});

export const {
  wsConnecting,
  wsConnected,
  wsDisconnected,
  wsError,
  wsMessage,
  clearWSError,
} = websocketSlice.actions;

export default websocketSlice.reducer; 