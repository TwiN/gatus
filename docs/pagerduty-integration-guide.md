# Instructions

* Please copy this template (copy either the markdown by clicking **Raw** or the copy directly from this preview) and use it to create your guide in your preferred medium. 
* This template includes information required in all PagerDuty integration guides.
* Template instructions in ***bold italics*** are intended to provide guidance or examples, and should be deleted and/or replaced once you’ve added your content.
* If your integration does not follow the same flow as what we’ve provided below (e.g. steps begin in your tool’s UI as opposed to the PagerDuty UI, etc.), feel free to make the changes you need to reflect the flow of the integration.
* Please read through our Writing and Style Guidelines below before starting your draft.

# Writing and Style Guidelines

## Detailed, Explicit Instructions
All steps to completing the integration should live in this guide. It's always good to err on the side of too much information, even if you think something is obvious. By writing your instructions as if the reader has had zero experience with any of the content, you can proactively anticipate any customer questions and greatly relieve Support efforts. 

**Example**:
* **Don't**: "Find your ClientID and paste it into this field."
* **Do**: "Navigate to **Account Settings** in the system menu and copy your **Client ID**. Next, navigate back to the **Configuration** page and paste it in the **ID** field." 

## Calls to Action

Most calls to action include clickable objects or fields, which you should highlight with **bold text**. This helps the reader follow along in the instructions and denotes when they should be taking action in the UI. 

**Examples**:
* "Navigate to the **Configuration** menu and select **Users**."
* "Paste the **Integration Key** into the **Token** field"

## Actionable Steps
Summaries before your content may work well when giving a talk or presenting to a targeted crowd, but not in documentation that users are more likely to skim hoping for quick answers. TL;DR: Don't include sentences that just state what you plan on writing about. If you feel you need to add more information that contextualizes what the reader is configuring, include it within the steps, or in a quick summary after them. 

**Example**
* **Don't**: "In this procedure we will be creating a Topic and a Subscription that will then allow you to create messages that trigger PagerDuty incidents..." etc.
* **Do**: "1. Navigate to the **Topics** tab and click **Create Topic**. 2. Enter a **Name**, configure your details and click **Save Topic**. 3. Next, navigate to the **Subscriptions** tab and click **Create Subscription**. Enter a **Name**, select the **Topic** created in step 1 and paste your PagerDuty **Integration Key** in the **Token** field. Click **Save Subscription**. You have now created a subscription that references your PagerDuty endpoint. When you publish a direct message to your Topic, it will trigger PagerDuty incidents."

## Use Active Voice
The active voice ensures that your writing is clear, concise and engaging. The [passive voice](https://webapps.towson.edu/ows/activepass.htm) uses more words, can sound vague and should be avoided like a [zombie plague](https://www.grammarly.com/blog/a-scary-easy-way-to-help-you-find-passive-voice/) (rhyme intended).

**Example**
* **Do**: "Users can follow incidents and escalations in real-time in Hungrycat’s event stream."
* **Don't**: "Incidents and escalations can be followed in real-time by users in Hungrycat’s event stream."

## Media
* At PagerDuty, we use the Preview tool that comes standard on macOS. Type **⌘ + ⇧ + A** or click **View** > **Show Markup Toolbar** to annotate images with arrows, rectangles and text.
* Only include screenshots that are **absolutely necessary**, so that you have less images to continually update when UI changes, etc. We usually only include screenshots when objects in the UI are small or harder to find. 
* Ensure that you've obfuscated all sensitive information in your screenshots (e.g., personal account information, integration keys, etc.,) by covering with fake data or an image blur tool. 

^^^ Note: Once you have completed your guide, please delete this section. ^^^
----






# PagerDuty + Gatus Integration Benefits
- Notify on-call responders based on alerts sent from Gatus.
- Incidents will automatically resolve in PagerDuty when the service that caused the incident in Gatus returns to a healthy state.


# How it Works
- Services that do not meet the user-specified conditions and that are configured with alerts of type `pagerduty` will trigger a new incident on the corresponding PagerDuty service when the alert's defined `failure-threshold` has been reached.
- Once the unhealthy services have returned to a healthy state for the number of executions defined in `success-threshold`, the previously triggered incident will be automatically resolved.


# Requirements
- PagerDuty integrations require an Admin base role for account authorization. If you do not have this role, please reach out to an Admin or Account Owner within your organization to configure the integration.


# Support

If you need help with this integration, please create an issue at https://github.com/TwinProduction/gatus/issues


# Integration Walkthrough
## In PagerDuty
### Integrating With a PagerDuty Service
1. From the **Configuration** menu, select **Services**.
2. There are two ways to add an integration to a service:
   * **If you are adding your integration to an existing service**: Click the **name** of the service you want to add the integration to. Then, select the **Integrations** tab and click the **New Integration** button.
   * **If you are creating a new service for your integration**: Please read our documentation in section [Configuring Services and Integrations](https://support.pagerduty.com/docs/services-and-integrations#section-configuring-services-and-integrations) and follow the steps outlined in the [Create a New Service](https://support.pagerduty.com/docs/services-and-integrations#section-create-a-new-service) section, selecting **Gatus** as the **Integration Type** in step 4. Continue with the In Gatus section (below) once you have finished these steps.
3. Enter an **Integration Name** in the format `monitoring-tool-service-name` (e.g. `Gatus-Shopping-Cart`) and select **Gatus** from the Integration Type menu.
4. Click the **Add Integration** button to save your new integration. You will be redirected to the Integrations tab for your service.
5. An **Integration Key** will be generated on this screen. Keep this key saved in a safe place, as it will be used when you configure the integration with **Gatus** in the next section.
![PagerDuty Integration Key](../.github/assets/pagerduty-integration-key.png)

## In Gatus
In your configuration file, you must first specify the integration key in `alerting.pagerduty`, like so:

```yaml
alerting:
  pagerduty: "********************************"
```

You can now add alerts of type `pagerduty` in the services you've defined, like so:

```yaml
services:
  - name: twinnation
    interval: 30s
    url: "https://twinnation.org/health"
    alerts:
      - type: pagerduty
        enabled: true
        failure-threshold: 3
        success-threshold: 5
        description: "healthcheck failed 3 times in a row"
        send-on-resolved: true
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"
      - "[RESPONSE_TIME] < 300"
```

The sample above will do the following:
- Send a request to the **https://twinnation.org/health** (`services[].url`) specified every **30s** (`services[].interval`)
- Evaluate the conditions that mark this service as "healthy"
- **If all conditions are not met 3 (`services[].alerts[].failure-threshold`) times in a row**: Gatus will create a new incident
- **If, after an incident has been triggered, all conditions are met 5 (`services[].alerts[].success-threshold`) times in a row _AND_ `services[].alerts[].send-on-resolved` is set to `true`**: Gatus will resolve the triggered incident

It is highly recommended to set `services[].alerts[].send-on-resolved` to true for alerts of type `pagerduty`.

# How to Uninstall
1. Navigate to the PagerDuty service you'd like to uninstall the Gatus integration from
2. Click on the **Integration** tab
3. Click on the **Gatus** integration
4. Click on **Delete Integration**

While the above will prevent incidents from being created, you are also highly encouraged to disable the alerts
in your Gatus configuration files or simply remove the integration key from the configuration file.
