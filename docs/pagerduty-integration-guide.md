# PagerDuty + Gatus Integration Benefits
- Notify on-call responders based on alerts sent from Gatus.
- Incidents will automatically resolve in PagerDuty when the endpoint that caused the incident in Gatus returns to a healthy state.


# How it Works
- Endpoints that do not meet the user-specified conditions and that are configured with alerts of type `pagerduty` will trigger a new incident on the corresponding PagerDuty service when the alert's defined `failure-threshold` has been reached.
- Once the unhealthy endpoints have returned to a healthy state for the number of executions defined in `success-threshold`, the previously triggered incident will be automatically resolved.


# Requirements
- PagerDuty integrations require an Admin base role for account authorization. If you do not have this role, please reach out to an Admin or Account Owner within your organization to configure the integration.


# Support
If you need help with this integration, please create an issue at https://github.com/TwiN/gatus/issues


# Integration Walkthrough
## In PagerDuty
### Integrating With a PagerDuty Service
1. From the **Configuration** menu, select **Services**.
2. There are two ways to add an integration to a service:
   * **If you are adding your integration to an existing service**: Click the **name** of the service you want to add the integration to. Then, select the **Integrations** tab and click the **New Integration** button.
   * **If you are creating a new service for your integration**: Please read our documentation in section [Configuring Services and Integrations](https://support.pagerduty.com/docs/services-and-integrations#section-configuring-services-and-integrations) and follow the steps outlined in the [Create a New Service](https://support.pagerduty.com/docs/services-and-integrations#section-create-a-new-service) section, selecting **Gatus** as the **Integration Type** in step 4. Continue with the In Gatus section (below) once you have finished these steps.
3. Enter an **Integration Name** in the format `gatus-service-name` (e.g. `Gatus-Shopping-Cart`) and select **Gatus** from the Integration Type menu.
4. Click the **Add Integration** button to save your new integration. You will be redirected to the Integrations tab for your service.
5. An **Integration Key** will be generated on this screen. Keep this key saved in a safe place, as it will be used when you configure the integration with **Gatus** in the next section.
![PagerDuty Integration Key](https://raw.githubusercontent.com/TwiN/gatus/master/.github/assets/pagerduty-integration-key.png)


## In Gatus
In your configuration file, you must first specify the integration key at `alerting.pagerduty.integration-key`, like so:
```yaml
alerting:
  pagerduty: 
    integration-key: "********************************"
```
You can now add alerts of type `pagerduty` in the endpoint you've defined, like so:
```yaml
endpoints:
  - name: website
    interval: 30s
    url: "https://twin.sh/health"
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
- Send a request to the `https://twin.sh/health` (`endpoints[].url`) specified every **30s** (`endpoints[].interval`)
- Evaluate the conditions to determine whether the endpoint is "healthy" or not
- **If all conditions are not met 3 (`endpoints[].alerts[].failure-threshold`) times in a row**: Gatus will create a new incident
- **If, after an incident has been triggered, all conditions are met 5 (`endpoints[].alerts[].success-threshold`) times in a row _AND_ `endpoints[].alerts[].send-on-resolved` is set to `true`**: Gatus will resolve the triggered incident

It is highly recommended to set `endpoints[].alerts[].send-on-resolved` to true for alerts of type `pagerduty`.


# How to Uninstall
1. Navigate to the PagerDuty service you'd like to uninstall the Gatus integration from
2. Click on the **Integration** tab
3. Click on the **Gatus** integration
4. Click on **Delete Integration**

While the above will prevent incidents from being created, you are also highly encouraged to disable the alerts
in your Gatus configuration files or simply remove the integration key from the configuration file.
