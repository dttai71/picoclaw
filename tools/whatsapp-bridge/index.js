const { default: makeWASocket, useMultiFileAuthState, DisconnectReason } = require('@whiskeysockets/baileys');
const { WebSocketServer } = require('ws');
const qrcode = require('qrcode-terminal');

const WS_PORT = parseInt(process.env.BRIDGE_PORT || '3001', 10);
const PING_INTERVAL = 54000;
const PROTOCOL_VERSION = '1';

let picoClawSocket = null;
let waSocket = null;

// --- WebSocket Server (for PicoClaw connection) ---

const wss = new WebSocketServer({ port: WS_PORT });

wss.on('listening', () => {
  console.log(`[bridge] WebSocket server listening on ws://0.0.0.0:${WS_PORT}`);
  console.log('[bridge] Waiting for PicoClaw to connect...');
});

wss.on('connection', (ws) => {
  console.log('[bridge] PicoClaw connected');
  picoClawSocket = ws;

  const pingTimer = setInterval(() => {
    if (ws.readyState === ws.OPEN) ws.ping();
  }, PING_INTERVAL);

  ws.on('message', (data) => {
    try {
      const msg = JSON.parse(data.toString());
      if (msg.type === 'message' && msg.to && msg.content) {
        sendWhatsAppMessage(msg.to, msg.content);
      }
    } catch (err) {
      console.error('[bridge] Failed to parse message from PicoClaw:', err.message);
    }
  });

  ws.on('close', () => {
    console.log('[bridge] PicoClaw disconnected');
    clearInterval(pingTimer);
    picoClawSocket = null;
  });

  ws.on('error', (err) => {
    console.error('[bridge] WebSocket error:', err.message);
  });

  // Send current WhatsApp status
  sendToPicoClaw({
    v: PROTOCOL_VERSION,
    type: 'status',
    timestamp: Date.now(),
    status: waSocket ? 'connected' : 'disconnected',
  });
});

// --- Send to PicoClaw ---

function sendToPicoClaw(obj) {
  if (picoClawSocket && picoClawSocket.readyState === picoClawSocket.OPEN) {
    picoClawSocket.send(JSON.stringify(obj));
  }
}

// --- WhatsApp Connection ---

async function connectWhatsApp() {
  const { state, saveCreds } = await useMultiFileAuthState('./auth_store');

  waSocket = makeWASocket({
    auth: state,
    printQRInTerminal: false,
  });

  waSocket.ev.on('creds.update', saveCreds);

  waSocket.ev.on('connection.update', (update) => {
    const { connection, lastDisconnect, qr } = update;

    if (qr) {
      console.log('\n[whatsapp] Scan QR code with your WhatsApp:');
      qrcode.generate(qr, { small: true });

      sendToPicoClaw({
        v: PROTOCOL_VERSION,
        type: 'status',
        timestamp: Date.now(),
        status: 'qr_required',
        qr: qr,
      });
    }

    if (connection === 'close') {
      const statusCode = lastDisconnect?.error?.output?.statusCode;
      const shouldReconnect = statusCode !== DisconnectReason.loggedOut;

      console.log(`[whatsapp] Disconnected (code: ${statusCode})`);

      sendToPicoClaw({
        v: PROTOCOL_VERSION,
        type: statusCode === DisconnectReason.loggedOut ? 'error' : 'status',
        timestamp: Date.now(),
        code: statusCode === DisconnectReason.loggedOut ? 'AUTH_FAILED' : 'WHATSAPP_DISCONNECTED',
        status: 'disconnected',
        message: `Disconnected (code: ${statusCode})`,
        retry_after: shouldReconnect ? 5 : 0,
      });

      if (shouldReconnect) {
        console.log('[whatsapp] Reconnecting in 5s...');
        setTimeout(connectWhatsApp, 5000);
      } else {
        console.log('[whatsapp] Logged out. Delete ./auth_store and restart to re-authenticate.');
      }
    }

    if (connection === 'open') {
      console.log('[whatsapp] Connected successfully!');
      sendToPicoClaw({
        v: PROTOCOL_VERSION,
        type: 'status',
        timestamp: Date.now(),
        status: 'connected',
      });
    }
  });

  waSocket.ev.on('messages.upsert', ({ messages, type }) => {
    if (type !== 'notify') return;

    for (const msg of messages) {
      if (msg.key.fromMe) continue;
      if (!msg.message) continue;

      const chatId = msg.key.remoteJid;
      const senderId = msg.key.participant || chatId;
      const pushName = msg.pushName || '';

      // Extract text content
      let content = '';
      if (msg.message.conversation) {
        content = msg.message.conversation;
      } else if (msg.message.extendedTextMessage) {
        content = msg.message.extendedTextMessage.text || '';
      } else if (msg.message.imageMessage) {
        content = msg.message.imageMessage.caption || '[image]';
      } else if (msg.message.videoMessage) {
        content = msg.message.videoMessage.caption || '[video]';
      } else if (msg.message.documentMessage) {
        content = '[document]';
      } else if (msg.message.audioMessage) {
        content = '[audio]';
      } else if (msg.message.stickerMessage) {
        content = '[sticker]';
      }

      if (!content) continue;

      // Clean phone number (remove @s.whatsapp.net)
      const cleanId = (id) => id.replace(/@s\.whatsapp\.net$/, '').replace(/@g\.us$/, '');

      sendToPicoClaw({
        v: PROTOCOL_VERSION,
        type: 'message',
        timestamp: Date.now(),
        id: msg.key.id,
        from: cleanId(senderId),
        chat: cleanId(chatId),
        content: content,
        from_name: pushName,
      });

      console.log(`[whatsapp] Message from ${pushName || cleanId(senderId)}: ${content.substring(0, 50)}...`);
    }
  });
}

// --- Send WhatsApp Message ---

async function sendWhatsAppMessage(to, content) {
  if (!waSocket) {
    console.error('[whatsapp] Not connected, cannot send message');
    return;
  }

  // Add WhatsApp JID suffix if not present
  const jid = to.includes('@') ? to : `${to}@s.whatsapp.net`;

  try {
    await waSocket.sendMessage(jid, { text: content });
    console.log(`[whatsapp] Sent message to ${to}: ${content.substring(0, 50)}...`);
  } catch (err) {
    console.error(`[whatsapp] Failed to send message: ${err.message}`);
    sendToPicoClaw({
      v: PROTOCOL_VERSION,
      type: 'error',
      timestamp: Date.now(),
      code: 'SEND_FAILED',
      message: err.message,
    });
  }
}

// --- Start ---

console.log('[bridge] PicoClaw WhatsApp Bridge v1.0');
connectWhatsApp().catch(err => {
  console.error('[whatsapp] Fatal error:', err);
  process.exit(1);
});
