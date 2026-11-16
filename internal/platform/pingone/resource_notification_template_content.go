package pingone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/patrickcping/pingone-go-sdk-v2/management"
)

// pingone_notification_template_content is a child of the notification
// templates family: contents are listed per template name, so listing
// requires scanning all EnumTemplateName values (list-then-scan over the
// fixed template enum, not a parent resource list).
//
// Bootstrap filtering (skill Phase 1a.5): every environment ships
// predefined default contents (Default == true) whose text is read-only —
// the provider's pingone_notification_template_content errors when
// importing them. They are deterministic, structurally unmanageable, and
// present in every environment, so the handler drops them before
// processing and keeps only environment-authored custom contents.
func init() {
	registerResource("pingone_notification_template_content", resourceHandler{
		list: listNotificationTemplateContents,
		get:  getNotificationTemplateContent,
	})
}

// notificationTemplateContentData projects the TemplateContent union
// (email/push/sms/voice) onto the flat source paths declared in
// notification_template_content.yaml. TemplateName is not part of the API
// response envelope (it is the request path), so it rides along from the
// list/get context; same for the environment ID.
type notificationTemplateContentData struct {
	ID             string
	EnvironmentID  string
	TemplateName   string
	Locale         string
	DeliveryMethod string
	Default        bool
	Variant        string

	EmailBody           string
	EmailSubject        string
	EmailFromAddress    string
	EmailFromName       string
	EmailReplyToAddress string
	EmailReplyToName    string
	EmailCharset        string
	EmailContentType    string

	PushTitle string
	PushBody  string

	SMSContent string
	SMSSender  string

	VoiceContent string
	VoiceType    string
}

// projectTemplateContent unwraps the union and flattens the populated
// variant's fields. Returns nil when the union member is unrecognized.
func projectTemplateContent(envID, templateName string, content *management.TemplateContent) *notificationTemplateContentData {
	switch v := content.GetActualInstance().(type) {
	case *management.TemplateContentEmail:
		d := commonTemplateContent(envID, templateName, v.GetId(), v.Locale, string(v.DeliveryMethod), v.GetDefault(), v.Variant)
		d.EmailBody = v.Body
		d.EmailSubject = v.GetSubject()
		if v.From != nil {
			d.EmailFromAddress = v.From.GetAddress()
			d.EmailFromName = v.From.GetName()
		}
		if v.ReplyTo != nil {
			d.EmailReplyToAddress = v.ReplyTo.GetAddress()
			d.EmailReplyToName = v.ReplyTo.GetName()
		}
		d.EmailCharset = v.GetCharset()
		d.EmailContentType = v.GetEmailContentType()
		return d
	case *management.TemplateContentPush:
		d := commonTemplateContent(envID, templateName, v.GetId(), v.Locale, string(v.DeliveryMethod), v.GetDefault(), v.Variant)
		d.PushTitle = v.Title
		d.PushBody = v.Body
		return d
	case *management.TemplateContentSMS:
		d := commonTemplateContent(envID, templateName, v.GetId(), v.Locale, string(v.DeliveryMethod), v.GetDefault(), v.Variant)
		d.SMSContent = v.Content
		d.SMSSender = v.GetSender()
		return d
	case *management.TemplateContentVoice:
		d := commonTemplateContent(envID, templateName, v.GetId(), v.Locale, string(v.DeliveryMethod), v.GetDefault(), v.Variant)
		d.VoiceContent = v.Content
		d.VoiceType = v.GetVoice()
		return d
	default:
		return nil
	}
}

func commonTemplateContent(envID, templateName, id, locale, deliveryMethod string, def bool, variant *string) *notificationTemplateContentData {
	d := &notificationTemplateContentData{
		ID:             id,
		EnvironmentID:  envID,
		TemplateName:   templateName,
		Locale:         locale,
		DeliveryMethod: deliveryMethod,
		Default:        def,
	}
	if variant != nil {
		d.Variant = *variant
	}
	return d
}

// allTemplateNames enumerates the 16 EnumTemplateName values the SDK
// declares, in declaration order.
func allTemplateNames() []management.EnumTemplateName {
	return []management.EnumTemplateName{
		management.ENUMTEMPLATENAME_CREDENTIAL_ISSUED,
		management.ENUMTEMPLATENAME_CREDENTIAL_REVOKED,
		management.ENUMTEMPLATENAME_CREDENTIAL_UPDATED,
		management.ENUMTEMPLATENAME_CREDENTIAL_VERIFICATION,
		management.ENUMTEMPLATENAME_DEVICE_PAIRING,
		management.ENUMTEMPLATENAME_DIGITAL_WALLET_PAIRING,
		management.ENUMTEMPLATENAME_EMAIL_PHONE_VERIFICATION,
		management.ENUMTEMPLATENAME_EMAIL_VERIFICATION_ADMIN,
		management.ENUMTEMPLATENAME_EMAIL_VERIFICATION_USER,
		management.ENUMTEMPLATENAME_GENERAL,
		management.ENUMTEMPLATENAME_ID_VERIFICATION,
		management.ENUMTEMPLATENAME_NEW_DEVICE_PAIRED,
		management.ENUMTEMPLATENAME_RECOVERY_CODE_TEMPLATE,
		management.ENUMTEMPLATENAME_STRONG_AUTHENTICATION,
		management.ENUMTEMPLATENAME_TRANSACTION,
		management.ENUMTEMPLATENAME_VERIFICATION_CODE_TEMPLATE,
	}
}

// listNotificationTemplateContents scans contents for every template name,
// unwraps the union, and keeps only custom (non-default) contents.
//
// Raw HTTP instead of the SDK iterator: several templates answer 500
// UNEXPECTED_ERROR in environments without their service provisioned (the
// credential_* templates are reliably affected), and the SDK's transport
// retries every 500 ten times with 30s backoff — ~5 minutes per failing
// template, ~30 minutes of pure backoff per export. A single-attempt GET
// per template keeps the scan fast and treats the deterministic absences
// as empty results. Same rationale as the flow handler's raw-HTTP list
// (resource_flow.go, listFlowIDs).
func listNotificationTemplateContents(ctx context.Context, c *Client, _ string) ([]interface{}, error) {
	cfg := c.apiClient.GetConfig()
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "https"
	}

	envID := c.environmentID.String()
	var result []interface{}
	for _, tn := range allTemplateNames() {
		contents, err := fetchTemplateContents(ctx, scheme, cfg.Host, envID, string(tn), cfg.HTTPClient)
		if err != nil {
			return nil, err
		}
		for i := range contents {
			projected := projectTemplateContent(envID, string(tn), &contents[i])
			if projected == nil {
				continue
			}
			if projected.Default {
				continue // predefined read-only content
			}
			result = append(result, projected)
		}
	}
	return result, nil
}

// fetchTemplateContents GETs one template's contents with a single attempt.
// 404 (notifications module not provisioned) and 500 UNEXPECTED_ERROR
// (service-dependent templates in environments without that service) are
// deterministic absences: they return (nil, nil) rather than an error.
// Non-paginated assumption: contents lists are small; if _links.pagination
// ever appears the caller can extend this to follow next links.
func fetchTemplateContents(ctx context.Context, scheme, host, envID, templateName string, httpClient *http.Client) ([]management.TemplateContent, error) {
	reqURL := fmt.Sprintf("%s://%s/v1/environments/%s/templates/%s/contents",
		scheme, host, envID, url.PathEscape(templateName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create template contents request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute template contents request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read template contents response: %w", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		// populated below
	case http.StatusNotFound, http.StatusInternalServerError:
		// deterministic absence (see listNotificationTemplateContents)
		return nil, nil
	default:
		return nil, fmt.Errorf("list template contents for %s: unexpected status %d: %s", templateName, resp.StatusCode, string(body))
	}

	var parsed struct {
		Embedded struct {
			Contents []management.TemplateContent `json:"contents"`
		} `json:"_embedded"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse template contents response for %s: %w", templateName, err)
	}
	return parsed.Embedded.Contents, nil
}

// getNotificationTemplateContent reads one content. resourceID is a
// composite "templateName/contentID" string.
func getNotificationTemplateContent(ctx context.Context, c *Client, _ string, resourceID string) (interface{}, error) {
	parts := splitTwo(resourceID, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("get notification template content: resourceID must be templateName/contentID, got: %s", resourceID)
	}
	templateName, contentID := parts[0], parts[1]
	mgmt, err := c.management(ctx)
	if err != nil {
		return nil, err
	}
	content, _, err := mgmt.NotificationsTemplatesApi.ReadOneContent(ctx, c.environmentID.String(), management.EnumTemplateName(templateName), contentID).Execute()
	if err != nil {
		return nil, fmt.Errorf("get notification template content: %w", err)
	}
	projected := projectTemplateContent(c.environmentID.String(), templateName, content)
	if projected == nil {
		return nil, fmt.Errorf("get notification template content: unrecognized content union member for %s", contentID)
	}
	if projected.Default {
		return nil, fmt.Errorf("get notification template content: %s is a predefined default content and cannot be managed", contentID)
	}
	return projected, nil
}

// splitTwo splits s on the first occurrence of sep. Mirrors the composite-ID
// parsing in the sibling child-resource handlers (strings.SplitN semantics).
func splitTwo(s, sep string) []string {
	return strings.SplitN(s, sep, 2)
}
