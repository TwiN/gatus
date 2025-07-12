import {createRouter, createWebHistory} from 'vue-router'
import Home from '@/views/Home'
import Details from "@/views/Details";
import EndpointStatuses from '@/components/EndpointStatuses'

const routes = [
    {
        path: '/',
        name: 'Home',
        component: Home,
    },
    {
        path: '/endpoints/:key',
        name: 'Details',
        component: Details,
    },
    {
        path: '/endpoints-list',
        name: 'EndpointStatuses',
        component: EndpointStatuses,
    },
];

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
});

export default router;
