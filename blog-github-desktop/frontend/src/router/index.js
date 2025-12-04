import { createRouter, createWebHashHistory } from "vue-router";
import PostListView from "../views/PostListView.vue";
import PostDetailView from "../views/PostDetailView.vue";

const routes = [
  {
    path: "/",
    name: "home",
    component: PostListView,
  },
  {
    path: "/post/:slug",
    name: "post-detail",
    component: PostDetailView,
    props: true,
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
