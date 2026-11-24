import {createRouter, createWebHistory} from 'vue-router'
import Home from '@/views/Home'
import EndpointDetails from "@/views/EndpointDetails";
import SuiteDetails from '@/views/SuiteDetails';

const routes = [
    {
        path: '/',
        name: 'Home',
        component: Home
    },
    {
        path: '/endpoints/:key',
        name: 'EndpointDetails',
        component: EndpointDetails,
    },
    {
        path: '/suites/:key',
        name: 'SuiteDetails',
        component: SuiteDetails
    }
];

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
});

// Preserve valid ?theme= across SPA navigations (e.g. Home <-> Endpoint details)
router.beforeEach((to, from) => {
    const fromTheme = from.query.theme;
    if ((fromTheme === 'dark' || fromTheme === 'light') && !to.query.theme) {
        return { ...to, query: { ...to.query, theme: fromTheme } };
    }
});

export default router;
