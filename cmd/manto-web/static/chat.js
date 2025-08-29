const UI_CONFIG = {
  MAX_INPUT_HEIGHT: 120,
  MESSAGES: {
    SELECT_PROVIDER: "Please select a provider",
    INVALID_API_KEY: "Please enter a valid API key",
    INVALID_ANTHROPIC_KEY: "Anthropic API keys should start with 'sk-ant-'",
    NO_MODELS: "No models available for this API key",
    SELECT_MODEL: "Please select a model",
    MESSAGE_TOO_LONG: "Message too long",
    GENERIC_ERROR: "Something went wrong. Please try again.",
  },
};

const validate = {
  apiKey: (key, config) =>
    key?.trim().length >= (config?.validation?.minApiKeyLength || 10),
  anthropicKey: (key, config) =>
    key?.startsWith(config?.api?.anthropicKeyPrefix || "sk-ant-"),
  message: (msg, config) =>
    msg?.trim() && msg.length <= (config?.validation?.maxMessageLength || 4000),
  provider: (provider) => Boolean(provider),
  model: (model) => Boolean(model),
};

class ChatError extends Error {
  constructor(message, type = "GENERAL") {
    super(message);
    this.type = type;
  }
}

function handleError(error, context) {
  console.error(`[${context}]`, error);

  const userMessage =
    error instanceof ChatError
      ? error.message
      : UI_CONFIG.MESSAGES.GENERIC_ERROR;

  showUserError(userMessage);
}

function showUserError(message) {
  showValidationMessage(message);
}

function showValidationMessage(message, isApiKeyError = false) {
  const elementId = isApiKeyError
    ? "apiKeyValidationMessage"
    : "validationMessage";
  const validationElement = document.getElementById(elementId);
  if (!validationElement) return;

  validationElement.textContent = message;
  validationElement.style.display = "block";
}

function hideValidationMessage() {
  const providerValidation = document.getElementById("validationMessage");
  const apiKeyValidation = document.getElementById("apiKeyValidationMessage");

  if (providerValidation) providerValidation.style.display = "none";
  if (apiKeyValidation) apiKeyValidation.style.display = "none";
}

const ChatApp = {
  state: {
    apiKey: null,
    currentProvider: null,
    currentModel: null,
    config: null,
    conversationHistory: [],
    isGenerating: false,
  },

  elements: {},

  init() {
    this.cacheElements();

    if (!this.validateRequiredElements()) {
      console.error("Missing required DOM elements");
      return;
    }

    this.loadConfig();
    this.setupEventListeners();
    this.showSetup();
  },

  cacheElements() {
    this.elements = {
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
  },

  validateRequiredElements() {
    const required = ["setupModal", "chatMain", "messageInput"];
    return required.every((key) => this.elements[key]);
  },

  loadConfig() {
    if (window.MantoConfig?.providers) {
      this.state.config = window.MantoConfig;
      this.populateProviders();
    } else {
      console.warn("Config not found, using fallback");
      this.state.config = {
        providers: [
          {
            name: "anthropic",
            displayName: "Anthropic",
            apiEndpoint: "https://api.anthropic.com",
            apiVersion: "2023-06-01",
          },
        ],
      };
      this.populateProviders();
    }
  },

  populateProviders() {
    const select = this.elements.setupProvider;
    if (!select) return;

    select.innerHTML = '<option value="">Choose a provider...</option>';

    this.state.config.providers.forEach((provider) => {
      const option = document.createElement("option");
      option.value = provider.name;
      option.textContent = provider.displayName;
      option.dataset.defaultModel = provider.defaultModel;
      select.appendChild(option);
    });
  },

  updateModelSelector(providerName, models = []) {
    const select = this.elements.modelSelector;
    if (!select) return;

    const provider = this.state.config.providers.find(
      (p) => p.name === providerName
    );
    if (!provider) return;

    if (models.length === 0) {
      select.innerHTML = '<option value="">Select Model...</option>';
      return;
    }

    select.innerHTML = '<option value="">Select Model...</option>';

    models.forEach((model) => {
      const option = document.createElement("option");
      option.value = model.id;
      option.textContent = model.display_name;
      option.title = this.formatModelCreatedDate(model.created_at);
      select.appendChild(option);
    });

    if (models.length > 0) {
      this.setDefaultModel(models, select);
    }
  },

  formatModelCreatedDate(createdAt) {
    return `Created: ${new Date(createdAt).toLocaleDateString()}`;
  },

  setDefaultModel(models, select) {
    const preferredIndex = models.findIndex(
      (model) =>
        model.id.includes("haiku") ||
        model.display_name?.toLowerCase().includes("haiku")
    );

    if (preferredIndex >= 0) {
      select.selectedIndex = preferredIndex + 1;
      this.state.currentModel = models[preferredIndex].id;
    } else {
      select.selectedIndex = 1;
      this.state.currentModel = models[0].id;
    }
  },

  async fetchModels(providerName, apiKey) {
    const provider = this.state.config.providers.find(
      (p) => p.name === providerName
    );
    if (!provider) {
      throw new ChatError("Provider not found", "VALIDATION");
    }

    try {
      const response = await fetch("/api/models", {
        method: "GET",
        headers: {
          "x-api-key": apiKey,
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new ChatError(errorData.error || "Failed to fetch models", "API");
      }

      const data = await response.json();
      return data.data || [];
    } catch (error) {
      if (error instanceof ChatError) {
        throw error;
      }
      console.error("Error fetching models:", error);
      throw new ChatError("Network error while fetching models", "NETWORK");
    }
  },

  async sendMessageToApi(model, messages) {
    try {
      const requestBody = {
        model: model,
        messages: messages,
      };

      const response = await fetch("/api/messages", {
        method: "POST",
        headers: {
          "x-api-key": this.state.apiKey,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(requestBody),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new ChatError(errorData.error || "Failed to send message", "API");
      }

      const data = await response.json();
      return data;
    } catch (error) {
      if (error instanceof ChatError) {
        throw error;
      }
      console.error("Error sending message:", error);
      throw new ChatError("Network error while sending message", "NETWORK");
    }
  },

  setupEventListeners() {
    this.elements.setupForm?.addEventListener("submit", (e) =>
      this.handleSetup(e)
    );
    this.elements.chatInputForm?.addEventListener("submit", (e) =>
      this.handleMessage(e)
    );
    this.elements.messageInput?.addEventListener("input", () =>
      this.handleInputChange()
    );
    this.elements.messageInput?.addEventListener("keydown", (e) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        this.handleMessage(e);
      }
    });
    this.elements.apiKeyInput?.addEventListener("input", () =>
      hideValidationMessage()
    );
    this.elements.newChatBtn?.addEventListener("click", () => this.clearChat());
    this.elements.hideTips?.addEventListener("click", () =>
      this.hidePrivacyNotice()
    );
    this.elements.modelSelector?.addEventListener("change", (e) => {
      this.state.currentModel = e.target.value;
      this.clearChat();
    });
  },

  async handleSetup(e) {
    e.preventDefault();

    hideValidationMessage();

    const provider = this.elements.setupProvider.value;
    const key = this.elements.apiKeyInput.value.trim();

    if (!this.validateSetupInputs(provider, key)) {
      return;
    }

    const submitBtn = this.elements.setupForm.querySelector(
      'button[type="submit"]'
    );
    this.setSubmitButtonLoading(submitBtn, true);

    try {
      const models = await this.fetchModels(provider, key);

      if (models.length === 0) {
        showValidationMessage(UI_CONFIG.MESSAGES.NO_MODELS, true);
        return;
      }

      this.state.apiKey = key;
      this.state.currentProvider = provider;
      this.updateModelSelector(provider, models);
      this.hideSetup();
      this.clearChat();
    } catch (error) {
      handleError(error, "Setup");
    } finally {
      this.setSubmitButtonLoading(submitBtn, false);
    }
  },

  validateSetupInputs(provider, key) {
    if (!validate.provider(provider)) {
      showValidationMessage(UI_CONFIG.MESSAGES.SELECT_PROVIDER, false);
      return false;
    }

    if (!validate.apiKey(key, this.state.config)) {
      showValidationMessage(UI_CONFIG.MESSAGES.INVALID_API_KEY, true);
      return false;
    }

    if (
      provider === "anthropic" &&
      !validate.anthropicKey(key, this.state.config)
    ) {
      showValidationMessage(UI_CONFIG.MESSAGES.INVALID_ANTHROPIC_KEY, true);
      return false;
    }

    return true;
  },

  setSubmitButtonLoading(submitBtn, isLoading) {
    if (isLoading) {
      submitBtn.originalText = submitBtn.textContent;
      submitBtn.disabled = true;
      submitBtn.textContent = "Loading models...";
    } else {
      submitBtn.disabled = false;
      submitBtn.textContent = submitBtn.originalText || "Continue";
    }
  },

  async handleMessage(e) {
    e.preventDefault();

    if (!this.validateMessagePreconditions()) {
      return;
    }

    const message = this.prepareMessage();
    if (!message) return;

    this.updateUIForSending();
    const loadingMessage = this.showLoadingMessage();

    try {
      const response = await this.sendAndProcessMessage(message);
      this.handleSuccessfulResponse(response, loadingMessage);
    } catch (error) {
      this.handleMessageError(error, loadingMessage);
    } finally {
      this.resetUIAfterSending();
    }
  },

  validateMessagePreconditions() {
    if (!this.state.apiKey) {
      this.showSetup();
      return false;
    }

    if (this.state.isGenerating) {
      return false;
    }

    return true;
  },

  prepareMessage() {
    const message = this.elements.messageInput.value.trim();
    if (!message) return null;

    if (!validate.message(message, this.state.config)) {
      const maxLength = this.state.config?.validation?.maxMessageLength || 4000;
      showUserError(`Message too long (max ${maxLength} characters)`);
      return null;
    }

    const model = this.elements.modelSelector.value;
    if (!validate.model(model)) {
      showUserError(UI_CONFIG.MESSAGES.SELECT_MODEL);
      return null;
    }

    return { text: message, model };
  },

  updateUIForSending() {
    const message = this.elements.messageInput.value.trim();
    this.addMessage(message, true);
    this.state.conversationHistory.push({ role: "user", content: message });

    this.elements.messageInput.value = "";
    this.elements.messageInput.style.height = "auto";
    this.updateSendButton();

    this.state.isGenerating = true;
    this.elements.sendBtn.disabled = true;
    this.elements.sendBtn.classList.add("generating");
  },

  showLoadingMessage() {
    const loadingMessage = this.addMessage("Thinking...", false);
    loadingMessage.classList.add("loading", "ai");
    return loadingMessage;
  },

  async sendAndProcessMessage(messageData) {
    return await this.sendMessageToApi(
      messageData.model,
      this.state.conversationHistory
    );
  },

  handleSuccessfulResponse(response, loadingMessage) {
    loadingMessage.remove();

    if (response.content && response.content.length > 0) {
      const assistantText = response.content
        .filter((block) => block.type === "text")
        .map((block) => block.text)
        .join("");

      if (assistantText) {
        this.addMessage(assistantText, false);
        this.state.conversationHistory.push({
          role: "assistant",
          content: assistantText,
        });
      }
    }
  },

  handleMessageError(error, loadingMessage) {
    loadingMessage.remove();
    handleError(error, "Message");
    this.addMessage(`Error: ${error.message}`, false, true);
  },

  resetUIAfterSending() {
    this.state.isGenerating = false;
    this.elements.sendBtn.disabled = false;
    this.elements.sendBtn.classList.remove("generating");
    this.updateSendButton();
  },

  handleInputChange() {
    const input = this.elements.messageInput;

    input.style.height = "auto";
    input.style.height =
      Math.min(input.scrollHeight, UI_CONFIG.MAX_INPUT_HEIGHT) + "px";

    this.updateSendButton();
  },

  updateSendButton() {
    const message = this.elements.messageInput.value.trim();
    const maxLength = this.state.config?.validation?.maxMessageLength || 4000;
    const isValid = message && message.length <= maxLength;
    this.elements.sendBtn.disabled = !isValid;
  },

  formatMessage(text) {
    if (!text) return "";

    const rawHtml = marked.parse(text);

    const cleanHtml = DOMPurify.sanitize(rawHtml, {
      ALLOWED_TAGS: [
        "h1",
        "h2",
        "h3",
        "h4",
        "h5",
        "h6",
        "p",
        "br",
        "strong",
        "em",
        "code",
        "pre",
        "ul",
        "ol",
        "li",
        "blockquote",
        "table",
        "thead",
        "tbody",
        "tr",
        "th",
        "td",
      ],
      ALLOWED_ATTR: ["class", "align"],
    });

    return cleanHtml;
  },

  addMessage(content, isUser, isError = false) {
    this.hidePrivacyNotice();

    const message = document.createElement("div");
    const messageClasses = this.buildMessageClasses(isUser, isError);
    message.className = messageClasses;

    const formattedContent =
      isUser || isError ? content : this.formatMessage(content);
    const avatar = isUser ? "U" : "AI";

    message.innerHTML = `
      <div class="message-avatar">${avatar}</div>
      <div class="message-content">${formattedContent}</div>
    `;

    this.elements.chatMessages.appendChild(message);

    return message;
  },

  buildMessageClasses(isUser, isError) {
    const classes = ["message"];

    if (isUser) {
      classes.push("user");
    } else {
      classes.push("ai");
    }

    if (isError) {
      classes.push("error");
    }

    return classes.join(" ");
  },

  showSetup() {
    this.elements.setupModal.style.display = "flex";
    this.elements.chatMain.style.display = "none";
  },

  hideSetup() {
    this.elements.setupModal.style.display = "none";
    this.elements.chatMain.style.display = "flex";
  },

  showPrivacyNotice() {
    this.elements.privacyNotice.style.display = "flex";
    this.elements.messagesContainer.style.display = "none";
  },

  hidePrivacyNotice() {
    this.elements.privacyNotice.style.display = "none";
    this.elements.messagesContainer.style.display = "block";
  },

  clearChat() {
    this.state.conversationHistory = [];
    this.elements.chatMessages.innerHTML = "";
    this.showPrivacyNotice();
  },
};

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", () => ChatApp.init());
} else {
  ChatApp.init();
}
