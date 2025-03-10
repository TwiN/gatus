import {createRouter, createWebHistory} from 'vue-router'
import Home from '@/views/Home'
import Group from '@/views/Group'
import Groups from '@/views/Groups'
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
    {
        path: '/groups',
        name: 'Groups',
        component: Groups
    },
    {
        path: '/groups/:group',
        name: 'Group',
        component: Group
    },
];

const router = createRouter({
    history: createWebHistory(process.env.BASE_URL),
    routes
});

router.beforeEach((to, from, next) => {
    if (from.path) {
      sessionStorage.setItem('previousPage', from.fullPath);
    }
    next();
});

export default router;
