import React, { useEffect, useState } from 'react';
import Layout from '../../../../components/DashboardLayout/dashboardLayout';
import PostsList from '../../../../components/PostsTable/postsTable';

const Index = ({ auth, dispatch }) => {
    return (
        <Layout selectedCategory={"Posts"}>
            <h1>Posts</h1>
            <PostsList />
        </Layout>
    );
};

export default Index;