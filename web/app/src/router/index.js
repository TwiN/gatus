import {createRouter, createWebHistory} from 'vue-router'
import Home from '@/views/Home'
import Details from "@/views/Details";

const routes = [
    {
        path: '/',
        name: 'Home',
        component: Home
    },
    {
        path: '/endpoints/:key',
        name: 'Details',
        component: Details,
    },
    { // XXX: Remove in v4.0.0
        path: '/services/:key',
        redirect: {name: 'Details'}
    },
];

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
});

export default router;
