<template>
  <div id="results">
    <div v-if="!isTvMode">
      <slot v-for="endpointGroup in endpointGroups" :key="endpointGroup">
        <EndpointGroup :endpoints="endpointGroup.endpoints" :name="endpointGroup.name" @showTooltip="showTooltip" @toggleShowAverageResponseTime="toggleShowAverageResponseTime" :showAverageResponseTime="showAverageResponseTime" />
      </slot>
    </div>

    <div v-else-if="isTvMode && totalServicesCount === 0" class="tv-loading">
      <slot v-for="endpointGroup in endpointGroups" :key="endpointGroup">
        <EndpointGroup :endpoints="endpointGroup.endpoints" :name="endpointGroup.name" @showTooltip="showTooltip" @toggleShowAverageResponseTime="toggleShowAverageResponseTime" :showAverageResponseTime="showAverageResponseTime" />
      </slot>
    </div>

    <div v-else-if="isTvMode" class="aggregated-mode">
      <div class="global-status-container" :class="getGlobalStatusClass">
        <div class="status-left p-4">
          <h1 class="text-center font-bold mb-2" :style="{ fontSize: globalTitleFontSize }">
            System status
          </h1>
          <div class="text-center" :style="{ fontSize: globalStatusFontSize }">
            {{ getGlobalStatusText }}
          </div>

          <div v-if="currentIncident" class="incident-info">
            <div class="text-center font-semibold mb-3" :style="{ fontSize: incidentTitleFontSize }">
              Active incidents
            </div>

            <div class="main-incident mb-3 p-2 bg-black bg-opacity-20 rounded">
              <div class="text-center font-medium mb-1" :style="{ fontSize: incidentDetailsFontSize }">
                {{ currentIncident.groupNames }}
              </div>
              <div class="text-center" :style="{ fontSize: incidentDetailsFontSize }">
                Max. duration: {{ currentIncident.maxDuration }} min
              </div>
            </div>

            <div class="text-center" :style="{ fontSize: incidentDetailsFontSize }">
              Groups affected: {{ currentIncident.affectedServices }}
            </div>

          </div>
        </div>

        <div class="status-right p-4 flex items-center justify-center">
          <div class="current-global-status flex items-center justify-center" :class="getCurrentGlobalStatusClass">
            <span class="global-status-icon" :style="{ fontSize: globalStatusIconFontSize }">
              {{ getGlobalStatusIcon }}
            </span>
          </div>
        </div>
      </div>

      <div class="groups-grid" :class="getServicesGridClass" :style="getServicesGridStyle">
        <div v-for="group in endpointGroups" :key="group.name"
             class="group-card p-2 rounded-lg" :class="getGroupCardClass(group)">
          <div class="text-center flex flex-col items-center justify-center gap-4">
            <h3 class="font-semibold mb-1 truncate leading-tight" :style="{ fontSize: serviceCardFontSize }">
              {{ group.name === 'undefined' ? 'Common services' : group.name }}
            </h3>
            <div class="text-sm mb-1" :style="{ fontSize: groupCardStatusFontSize }">
              {{ getGroupStatusSummary(group) }}
            </div>

            <div class="periods-history flex justify-center mb-1">
              <div v-for="(periodStatus, index) in getGroupPeriodHistory(group)"
                   :key="index"
                   :class="getPeriodStatusClass(periodStatus)"
                   class="period-indicator">
              </div>
            </div>

            <div class="flex justify-center">
              <div :class="getGroupCurrentStatusClass(group)" class="group-current-status rounded-full flex items-center justify-center">
                <span class="status-icon" :style="{ fontSize: statusIconFontSize }">
                  {{ getGroupCurrentStatusIcon(group) }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import EndpointGroup from './EndpointGroup.vue';
import {helper} from "@/mixins/helper";

const statuses = {
  ALL_HEALTHY: 'all-healthy',
  ALL_DOWN: 'all-down',
  PARTIAL: 'partial',
  NO_DATA: 'no-data'
}

export default {
  name: 'Endpoints',
  components: {
    EndpointGroup
  },
  mixins: [helper],
  props: {
    showStatusOnHover: Boolean,
    endpointStatuses: Object,
    showAverageResponseTime: Boolean
  },
  emits: ['showTooltip', 'toggleShowAverageResponseTime'],
  computed: {
    isTvMode() {
      return this.$route?.query?.mode === 'tv';
    },
    totalServicesCount() {
      if (!this.endpointGroups || this.endpointGroups.length === 0) {
        return 0;
      }
      return this.getAllEndpoints.length;
    },

    serviceTitleFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.7)}px`;
      }
      return '16px';
    },
    groupTitleFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.9)}px`;
      }
      return '20px';
    },
    groupStatusFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.5)}px`;
      }
      return '12px';
    },
    statusIconFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.25)}px`;
      }
      return '8px';
    },
    currentStatusIconFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.35)}px`;
      }
      return '12px';
    },
    periodsCount() {
      const periodsParam = this.$route?.query?.periods;
      if (periodsParam) {
        const count = parseInt(periodsParam);
        return Math.max(1, Math.min(10, count));
      }
      return 3;
    },
    globalTitleFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 1)}px`;
      }
      return '30px';
    },
    globalStatusFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.8)}px`;
      }
      return '24px';
    },
    globalStatusIconFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 2.2)}px`;
      }
      return '16px';
    },
    incidentTitleFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 1)}px`;
      }
      return '18px';
    },
    incidentDetailsFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.8)}px`;
      }
      return '14px';
    },
    serviceCardFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.9)}px`;
      }
      return '14px';
    },
    groupCardStatusFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.6)}px`;
      }
      return '12px';
    },
    incidentSmallFontSize() {
      const fontSizeParam = this.$route?.query?.font_size;
      if (fontSizeParam) {
        return `${Math.round(fontSizeParam * 0.7)}px`;
      }
      return '12px';
    },
    getAllEndpoints() {
      return this.endpointGroups.map(g => g.endpoints).flat();
    },
    getGlobalStatusClass() {
      const allEndpoints = this.getAllEndpoints;
      const status = this.getGlobalStatus(allEndpoints);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return 'bg-green-800';
        case statuses.ALL_DOWN:
          return 'bg-red-800';
        case statuses.PARTIAL:
          return 'bg-yellow-800';
        default:
          return 'bg-gray-800';
      }
    },
    getGlobalStatusText() {
      const allEndpoints = this.getAllEndpoints;
      const status = this.getGlobalStatus(allEndpoints);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return 'All services is UP';
        case statuses.ALL_DOWN:
          return 'All services is DOWN';
        case statuses.PARTIAL:
          return `Up ${status.healthy} services from ${status.total}`;
        default:
          return 'No data';
      }
    },
    getGlobalStatusIcon() {
      const allEndpoints = this.getAllEndpoints;
      const status = this.getGlobalStatus(allEndpoints);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return '✓';
        case statuses.ALL_DOWN:
          return '✗';
        case statuses.PARTIAL:
          return '⚠';
        default:
          return '?';
      }
    },
    getCurrentGlobalStatusClass() {
      const allEndpoints = this.getAllEndpoints;
      const status = this.getGlobalStatus(allEndpoints);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return 'bg-green-600';
        case statuses.ALL_DOWN:
          return 'bg-red-600';
        case statuses.PARTIAL:
          return 'bg-yellow-600';
        default:
          return 'bg-gray-600';
      }
    },
    getServicesGridClass() {
      if (this.isAggregatedMode) {
        return `grid gap-4 w-full max-w-full`;
      }
      return '';
    },
    getServicesGridStyle() {
      if (this.isAggregatedMode) {
        const groupsCount = this.endpointGroups.length;
        const cols = this.columnsCount;
        const rows = Math.ceil(groupsCount / cols);

        return {
          gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))`,
          gridTemplateRows: `repeat(${rows}, auto)`
        };
      }
      return {};
    },
    currentIncident() {
      return this.calculateCurrentIncident();
    }
  },
  watch: {
    endpointStatuses: function () {
      this.process();
    }
  },
  data() {
    return {
      userClickedStatus: false,
      endpointGroups: []
    }
  },
  methods: {
    process() {
      let outputByGroup = {};
      for (let endpointStatusIndex in this.endpointStatuses) {
        let endpointStatus = this.endpointStatuses[endpointStatusIndex];
        if (!outputByGroup[endpointStatus.group] || outputByGroup[endpointStatus.group].length === 0) {
          outputByGroup[endpointStatus.group] = [];
        }
        outputByGroup[endpointStatus.group].push(endpointStatus);
      }
      let endpointGroups = [];
      for (let name in outputByGroup) {
        if (name !== 'undefined') {
          endpointGroups.push({name: name, endpoints: outputByGroup[name]})
        }
      }
      if (outputByGroup['undefined']) {
        endpointGroups.push({name: 'undefined', endpoints: outputByGroup['undefined']})
      }
      this.endpointGroups = endpointGroups;
    },
    showTooltip(result, event) {
      this.$emit('showTooltip', result, event);
    },
    toggleShowAverageResponseTime() {
      this.$emit('toggleShowAverageResponseTime');
    },
    getServiceCardClass(endpoint) {
      if (!endpoint.results || endpoint.results.length === 0) {
        return 'bg-gray-100 dark:bg-gray-700';
      }

      const currentResult = endpoint.results[endpoint.results.length - 1];
      if (currentResult.success) {
        return 'bg-green-100 dark:bg-green-900';
      } else {
        return 'bg-red-100 dark:bg-red-900';
      }
    },
    getLastResults(endpoint, count) {
      if (!endpoint.results || endpoint.results.length === 0) {
        return [];
      }
      const results = endpoint.results.slice(-(count + 1), -1);
      while (results.length < count) {
        results.unshift(null);
      }
      return results;
    },
    getCurrentResult(endpoint) {
      if (!endpoint.results || endpoint.results.length === 0) {
        return null;
      }
      return endpoint.results[endpoint.results.length - 1];
    },
    getCompactHealthcheckClass(result, isCurrent = false) {
      const baseClass = 'healthcheck-status rounded-full';
      const sizeClass = isCurrent ? 'w-5 h-5' : 'w-3 h-3';

      if (!result) {
        return `${baseClass} ${sizeClass} bg-gray-300 dark:bg-gray-600 border border-gray-400 dark:border-gray-500`;
      }

      const statusClass = result.success
        ? 'bg-green-600 dark:bg-green-700'
        : 'bg-red-600 dark:bg-red-700';
      return `${baseClass} ${sizeClass} ${statusClass}`;
    },
    getGroupStatus(endpointGroup) {
      let healthy = 0;
      let total = 0;
      let hasData = false;

      for (const endpoint of endpointGroup.endpoints) {
        if (endpoint.results && endpoint.results.length > 0) {
          hasData = true;
          total++;
          if (endpoint.results[endpoint.results.length - 1].success) {
            healthy++;
          }
        }
      }

      if (!hasData) return { status: statuses.NO_DATA, healthy: 0, total: 0 };
      if (healthy === total) return { status: statuses.ALL_HEALTHY, healthy, total };
      if (healthy === 0) return { status: statuses.ALL_DOWN, healthy, total };
      return { status: statuses.PARTIAL, healthy, total };
    },
    getGroupStatusClass(endpointGroup) {
      const status = this.getGroupStatus(endpointGroup);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return 'bg-green-800';
        case statuses.ALL_DOWN:
          return 'bg-red-800';
        case statuses.PARTIAL:
          return 'bg-yellow-800';
        default:
          return 'bg-gray-800';
      }
    },
    getGroupTextClass(endpointGroup) {
      const status = this.getGroupStatus(endpointGroup);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
        case statuses.ALL_DOWN:
        case statuses.NO_DATA:
          return 'text-white';
        case statuses.PARTIAL:
          return 'text-gray-900 dark:text-white'
        default:
          return 'text-white';
      }
    },
    getGroupStatusText(endpointGroup) {
      const status = this.getGroupStatus(endpointGroup);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return `All services is UP (${status.total})`;
        case statuses.ALL_DOWN:
          return `All services is DOWN (${status.total})`;
        case statuses.PARTIAL:
          return `Up ${status.healthy} services from ${status.total}`;
        default:
          return 'No data';
      }
    },
    getGlobalStatus(endpoints) {
      let healthy = 0;
      let total = 0;
      let hasData = false;

      for (const endpoint of endpoints) {
        if (endpoint.results && endpoint.results.length > 0) {
          hasData = true;
          total++;
          if (endpoint.results[endpoint.results.length - 1].success) {
            healthy++;
          }
        }
      }

      if (!hasData) return { status: statuses.NO_DATA, healthy: 0, total: 0 };
      if (healthy === total) return { status: statuses.ALL_HEALTHY, healthy, total };
      if (healthy === 0) return { status: statuses.ALL_DOWN, healthy, total };
      return { status: statuses.PARTIAL, healthy, total };
    },
    calculateCurrentIncident() {
      const groupIncidents = [];
      let totalAffectedGroups = 0;

      for (const group of this.endpointGroups) {
        const groupHistory = this.getGroupPeriodHistory(group);
        let currentIncidentDuration = 0;
        let hasCurrentIncident = false;

        for (let i = groupHistory.length - 1; i >= 0; i--) {
          if (groupHistory[i] === 'down' || groupHistory[i] === 'partial') {
            hasCurrentIncident = true;
            currentIncidentDuration++;
          } else {
            break;
          }
        }

        if (hasCurrentIncident) {
          totalAffectedGroups++;
          groupIncidents.push({
            groupName: group.name === 'undefined' ? 'Common services' : group.name,
            duration: currentIncidentDuration * 5,
            status: this.getGroupStatus(group).status
          });
        }
      }

      if (totalAffectedGroups === 0) {
        return null;
      }

      const maxDuration = Math.max(...groupIncidents.map(incident => incident.duration));
      const groupNames = groupIncidents.map(incident => incident.groupName).join(', ');

      return {
        maxDuration: maxDuration,
        affectedServices: totalAffectedGroups,
        groupNames: groupNames,
        allIncidents: groupIncidents
      };
    },
    getGroupCardClass(group) {
      const status = this.getGroupStatus(group);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return 'bg-green-100 dark:bg-green-900 border-green-500';
        case statuses.ALL_DOWN:
          return 'bg-red-100 dark:bg-red-900 border-red-500';
        case statuses.PARTIAL:
          return 'bg-yellow-100 dark:bg-yellow-900 border-yellow-500';
        default:
          return 'bg-gray-100 dark:bg-gray-700 border-gray-500';
      }
    },
    getGroupStatusSummary(group) {
      const status = this.getGroupStatus(group);
      return `${status.healthy}/${status.total} ${status.total > 1 ? 'services' : 'service'}`;
    },
    getGroupPeriodHistory(group) {
      const history = [];
      const maxPeriods = this.periodsCount;

      const endpoints = group.endpoints;

      for (let periodIndex = 0; periodIndex < maxPeriods; periodIndex++) {
        let healthyInPeriod = 0;
        let totalInPeriod = 0;

        for (const endpoint of endpoints) {
          if (!endpoint.results || endpoint.results.length === 0) continue;

          const resultIndex = endpoint.results.length - 1 - periodIndex;
          if (resultIndex >= 0) {
            totalInPeriod++;
            if (endpoint.results[resultIndex].success) {
              healthyInPeriod++;
            }
          }
        }

        if (totalInPeriod === 0) {
          history.unshift('no-data');
        } else if (healthyInPeriod === totalInPeriod) {
          history.unshift('healthy');
        } else if (healthyInPeriod === 0) {
          history.unshift('down');
        } else {
          history.unshift('partial');
        }
      }

      return history;
    },
    getPeriodStatusClass(periodStatus) {
      const baseClass = 'w-2 h-2 mx-0.5 rounded-sm';
      switch (periodStatus) {
        case 'healthy':
          return `${baseClass} bg-green-500`;
        case 'down':
          return `${baseClass} bg-red-500`;
        case 'partial':
          return `${baseClass} bg-yellow-500`;
        default:
          return `${baseClass} bg-gray-300 border border-gray-400`;
      }
    },
    getGroupCurrentStatusClass(group) {
      const status = this.getGroupStatus(group);
      const baseClass = 'w-6 h-6';
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return `${baseClass} bg-green-600`;
        case statuses.ALL_DOWN:
          return `${baseClass} bg-red-600`;
        case statuses.PARTIAL:
          return `${baseClass} bg-yellow-600`;
        default:
          return `${baseClass} bg-gray-600`;
      }
    },
    getGroupCurrentStatusIcon(group) {
      const status = this.getGroupStatus(group);
      switch (status.status) {
        case statuses.ALL_HEALTHY:
          return '✓';
        case statuses.ALL_DOWN:
          return '✗';
        case statuses.PARTIAL:
          return '⚠';
        default:
          return '❓';
      }
    }
  }
}
</script>

<style>
.endpoint-group-content > div:nth-child(1) {
  border-top-left-radius: 0;
  border-top-right-radius: 0;
}

.services-container {
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(0, 0, 0, 0.05);
}

.dark .services-container {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.service-row {
  min-height: 28px;
  line-height: 1.2;
}

.service-name {
  min-width: 0;
}

.service-status {
  flex-shrink: 0;
}

.healthcheck-mini {
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.current-mini {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2), 0 0 8px rgba(0, 0, 0, 0.1);
}

.status-icon {
  font-weight: bold;
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  line-height: 1;
  opacity: 0.95;
}

.status-icon.current {
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.4);
  opacity: 1;
}

.group-header {
  background: linear-gradient(135deg, var(--tw-bg-opacity) 0%, rgba(0,0,0,0.05) 100%);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

@media screen and (max-width: 1024px) {
  .status-right {
    display: none;
  }
}

@media screen and (min-width: 1920px) {
  .healthcheck-mini.current-mini {
    width: 3rem !important;
    height: 3rem !important;
  }

  .healthcheck-mini:not(.current-mini) {
    width: 2rem !important;
    height: 2rem !important;
  }

  .service-row {
    min-height: 32px;
  }

  .services-container {
    max-height: 88vh;
  }
}

.aggregated-mode {
  display: flex;
  flex-direction: column;
  height: 100vh;
  gap: 0.5rem;
  padding: 0.75rem;
  overflow: hidden;
}

.global-status-container {
  display: grid;
  grid-template-columns: 2fr 1fr;
  border-radius: 1rem;
  color: white;
  margin-bottom: 0.5rem;
}

.status-left {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.status-right {
  border-left: 1px solid rgba(255, 255, 255, 0.2);
}

.current-global-status {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  color: white;
  font-weight: bold;
}

.global-status-icon {
  font-size: 32px;
}

.incident-info {
  margin-top: 1rem;
  padding: 1rem;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 0.5rem;
}

.groups-grid {
  flex-grow: 1;
  overflow: hidden;
  display: grid;
  gap: 0.75rem;
  height: calc(100vh - 140px);
  align-content: start;
}

.group-card {
  background: white;
  border: 2px solid;
  padding: 1rem;
  transition: all 0.2s ease;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100%;
  min-height: 120px;
}

.group-card:hover {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.periods-history {
  gap: 2px;
  flex-wrap: wrap;
  max-width: 100%;
}

.period-indicator {
  transition: all 0.2s ease;
  flex-shrink: 0;
  width: 1.5rem !important;
  height: 0.75rem !important;
  margin: 0 1px !important;
}

.period-indicator:hover {
  transform: scale(1.2);
}

.group-current-status {
  transition: all 0.3s ease;
  height: 36px;
  width: 36px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.dark .group-card {
  background: #374151;
  color: white;
}

.group-card.bg-green-100 {
  background-color: #dcfce7;
}

.group-card.bg-red-100 {
  background-color: #fee2e2;
}

.group-card.bg-yellow-100 {
  background-color: #fef3c7;
}

.group-card.bg-gray-100 {
  background-color: #f3f4f6;
}

.dark .group-card.bg-green-100 {
  background-color: #14532d;
}

.dark .group-card.bg-red-100 {
  background-color: #7f1d1d;
}

.dark .group-card.bg-yellow-100 {
  background-color: #78350f;
}

.dark .group-card.bg-gray-100 {
  background-color: #374151;
}


@media screen and (max-width: 768px) {
  .aggregated-mode {
    padding: 0.5rem !important;
    height: auto !important;
    overflow: auto !important;
  }

  .global-status-container {
    min-height: 150px !important;
  }

  .status-left {
    padding: 1rem !important;
  }

  .status-right {
    display: none;
  }

  .group-current-status {
    width: 1.5rem !important;
    height: 1.5rem !important;
  }

  .periods-history {
    justify-content: center !important;
  }

  .period-indicator {
    width: 1rem !important;
    height: 0.5rem !important;
  }
}

@media screen and (max-width: 1024px) {
  .global-status-container {
    grid-template-columns: 1fr;
    text-align: center;
    min-height: 120px;
  }

  .status-right {
    border-left: none;
    border-top: 1px solid rgba(255, 255, 255, 0.2);
    padding-top: 1rem !important;
  }

  .aggregated-mode .groups-grid {
    grid-template-columns: repeat(2, 1fr) !important;
    height: calc(100vh - 160px) !important;
  }

  .incident-info {
    margin-top: 0.5rem !important;
    padding: 0.75rem !important;
  }
}

@media screen and (max-width: 768px) {
  .aggregated-mode .groups-grid {
    grid-template-columns: 1fr !important;
    height: auto !important;
  }

  .group-card {
    min-height: auto !important;
    height: auto !important;
    width: 100% !important;
  }
}
</style>
