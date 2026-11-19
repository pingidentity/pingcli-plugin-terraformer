package pingone

import (
	"testing"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationTemplateContentResourceRegistered(t *testing.T) {
	assert.True(t, isSupported("pingone_notification_template_content"))
}

func TestNotificationTemplateContentResourceHandlerFunctions(t *testing.T) {
	h, ok := resourceHandlers["pingone_notification_template_content"]
	assert.True(t, ok)
	assert.NotNil(t, h.list)
	assert.NotNil(t, h.get)
}

func TestProjectTemplateContent_EmailVariant(t *testing.T) {
	email := management.NewTemplateContentEmail("en", management.ENUMTEMPLATECONTENTDELIVERYMETHOD_EMAIL, "Body text")
	email.SetId("content-1")
	email.SetSubject("Subject line")
	from := management.NewTemplateContentEmailAllOfFrom()
	from.SetAddress("noreply@example.com")
	from.SetName("BX Retail")
	email.From = from

	content := management.TemplateContentEmailAsTemplateContent(email)
	d := projectTemplateContent("env-1", "strong_authentication", &content)

	require.NotNil(t, d)
	assert.Equal(t, "content-1", d.ID)
	assert.Equal(t, "strong_authentication", d.TemplateName)
	assert.Equal(t, "en", d.Locale)
	assert.Equal(t, "Body text", d.EmailBody)
	assert.Equal(t, "Subject line", d.EmailSubject)
	assert.Equal(t, "noreply@example.com", d.EmailFromAddress)
	assert.Equal(t, "BX Retail", d.EmailFromName)
	assert.Empty(t, d.PushTitle, "non-email groups stay empty")
}

func TestProjectTemplateContent_SMSVariant(t *testing.T) {
	sms := management.NewTemplateContentSMS("en", management.ENUMTEMPLATECONTENTDELIVERYMETHOD_SMS, "Your code")
	sms.SetId("content-2")
	content := management.TemplateContentSMSAsTemplateContent(sms)
	d := projectTemplateContent("env-1", "general", &content)

	require.NotNil(t, d)
	assert.Equal(t, "Your code", d.SMSContent)
	assert.Empty(t, d.EmailBody)
}

func TestProjectTemplateContent_VoiceVariant(t *testing.T) {
	voice := management.NewTemplateContentVoice("en", management.ENUMTEMPLATECONTENTDELIVERYMETHOD_VOICE, "Hello")
	voice.SetId("content-3")
	content := management.TemplateContentVoiceAsTemplateContent(voice)
	d := projectTemplateContent("env-1", "general", &content)

	require.NotNil(t, d)
	assert.Equal(t, "Hello", d.VoiceContent)
}

func TestGetNotificationTemplateContent_MalformedCompositeID(t *testing.T) {
	c := &Client{}
	result, err := getNotificationTemplateContent(testCtx(), c, "", "no-slash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resourceID must be templateName/contentID")
	assert.Nil(t, result)
}

func TestGetNotificationTemplateContent_ManagementClientUnavailable(t *testing.T) {
	c := &Client{}
	result, err := getNotificationTemplateContent(testCtx(), c, "", "general/content-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "management API client not configured")
	assert.Nil(t, result)
}

func TestAllTemplateNames_CoversSDKEnum(t *testing.T) {
	names := allTemplateNames()
	assert.Len(t, names, 16)
	seen := map[string]bool{}
	for _, n := range names {
		assert.NotEmpty(t, string(n))
		assert.False(t, seen[string(n)], "duplicate template name %s", n)
		seen[string(n)] = true
	}
}
