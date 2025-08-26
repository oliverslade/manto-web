let apiKey = null;
let currentProvider = null;
let config = null;

let elements = {};

function init() {
  // Cache DOM elements
  elements = {
    setupModal: document.getElementById("setupModal"),
    setupForm: document.getElementById("setupForm"),
    setupProvider: document.getElementById("setupProvider"),
    apiKeyInput: document.getElementById("apiKey"),
    modelSelector: document.getElementById("modelSelector"),
    chatMain: document.getElementById("chatMain"),
    privacyNotice: document.getElementById("privacyNotice"),
    messagesContainer: document.getElementById("messagesContainer"),
    chatMessages: document.getElementById("chatMessages"),
    messageInput: document.getElementById("messageInput"),
    sendBtn: document.getElementById("sendBtn"),
    chatInputForm: document.getElementById("chatInputForm"),
    newChatBtn: document.getElementById("newChatBtn"),
    hideTips: document.getElementById("hideTips"),
  };

  if (!elements.setupModal || !elements.chatMain || !elements.messageInput) {
    console.error("Missing required DOM elements");
    return;
  }

  loadConfig();
  setupEventListeners();
  showSetup();
}

function loadConfig() {
  if (window.MantoConfig?.providers) {
    config = window.MantoConfig;
    populateProviders();
  } else {
    console.warn("Config not found, using fallback");
    config = {
      providers: [
        {
          name: "anthropic",
          displayName: "Anthropic",
          defaultModel: "claude-3-5-haiku-latest",
        },
      ],
    };
    populateProviders();
  }
}

function populateProviders() {
  const select = elements.setupProvider;
  if (!select) return;

  select.innerHTML = '<option value="">Choose a provider...</option>';

  config.providers.forEach((provider) => {
    const option = document.createElement("option");
    option.value = provider.name;
    option.textContent = provider.displayName;
    option.dataset.defaultModel = provider.defaultModel;
    select.appendChild(option);
  });
}

function updateModelSelector(providerName) {
  const select = elements.modelSelector;
  if (!select) return;

  const provider = config.providers.find((p) => p.name === providerName);
  if (!provider) return;

  select.innerHTML = `<option value="${provider.defaultModel}" selected>
    ${provider.displayName} ${provider.defaultModel}
  </option>`;
}

function setupEventListeners() {
  elements.setupForm?.addEventListener("submit", handleSetup);

  elements.chatInputForm?.addEventListener("submit", handleMessage);

  elements.messageInput?.addEventListener("input", handleInputChange);
  elements.messageInput?.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleMessage(e);
    }
  });

  elements.newChatBtn?.addEventListener("click", clearChat);
  elements.hideTips?.addEventListener("click", hidePrivacyNotice);
  elements.modelSelector?.addEventListener("change", clearChat);
}

function handleSetup(e) {
  e.preventDefault();

  const provider = elements.setupProvider.value;
  const key = elements.apiKeyInput.value.trim();

  if (!provider) {
    alert("Please select a provider");
    return;
  }

  if (!key || key.length < 10) {
    alert("Please enter a valid API key");
    return;
  }

  if (provider === "anthropic" && !key.startsWith("sk-ant-")) {
    alert("Anthropic API keys should start with 'sk-ant-'");
    return;
  }

  apiKey = key;
  currentProvider = provider;
  updateModelSelector(provider);
  hideSetup();
  clearChat();
}

function handleMessage(e) {
  e.preventDefault();

  if (!apiKey) {
    showSetup();
    return;
  }

  const message = elements.messageInput.value.trim();
  if (!message) return;

  if (message.length > 4000) {
    alert("Message too long (max 4000 characters)");
    return;
  }

  addMessage(message, true);
  elements.messageInput.value = "";
  elements.messageInput.style.height = "auto";
  updateSendButton();

  // Demo response for now
  const model = elements.modelSelector.value;
  addMessage(
    `Demo response from ${model}. API integration coming soon!`,
    false
  );
}

function handleInputChange() {
  const input = elements.messageInput;

  input.style.height = "auto";
  input.style.height = Math.min(input.scrollHeight, 120) + "px";

  updateSendButton();
}

function updateSendButton() {
  const message = elements.messageInput.value.trim();
  elements.sendBtn.disabled = !message || message.length > 4000;
}

function addMessage(content, isUser) {
  hidePrivacyNotice();

  const message = document.createElement("div");
  message.className = `message ${isUser ? "user" : "ai"}`;

  message.innerHTML = `
    <div class="message-avatar">${isUser ? "U" : "AI"}</div>
    <div class="message-content">${content}</div>
  `;

  elements.chatMessages.appendChild(message);
  message.scrollIntoView({ behavior: "smooth", block: "end" });
}

function showSetup() {
  elements.setupModal.style.display = "flex";
  elements.chatMain.style.display = "none";
}

function hideSetup() {
  elements.setupModal.style.display = "none";
  elements.chatMain.style.display = "flex";
}

function showPrivacyNotice() {
  elements.privacyNotice.style.display = "flex";
  elements.messagesContainer.style.display = "none";
}

function hidePrivacyNotice() {
  elements.privacyNotice.style.display = "none";
  elements.messagesContainer.style.display = "block";
}

function clearChat() {
  elements.chatMessages.innerHTML = "";
  showPrivacyNotice();
}

// Initialize when DOM is ready
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", init);
} else {
  init();
}
