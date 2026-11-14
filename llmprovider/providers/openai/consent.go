package openai

// ConsentBackgroundUse is the id of the consent the ChatGPT plan's sign-in
// asks: that the app may also use the person's plan for background requests
// it starts on its own, which OpenAI's terms ask an app to obtain
// expressly (0075-MADR D1).
const ConsentBackgroundUse = "chatgpt_plan.background_use"

// consentBackgroundText is the consent's default wording: the notice and the
// question in one.
const consentBackgroundText = "You're using your ChatGPT plan: this app's requests count toward its usage, " +
	"which you can manage on the usage page. " +
	"Allow this app to also use your plan for background requests it starts on its own?"
