import Layout from '../components/PublicLayout/publicLayout';
import PostsContainer from '../components/Posts/postsContainer';
import fetch from 'isomorphic-unfetch';
import {wrapper, State} from '../redux/store';
import { fetchPosts as fetchPostsAction } from '../redux/fetchPosts/actions';

const Index = ({initialPosts, time}) => {
    const timeString = new Date(time).toLocaleTimeString();
    return (
        <Layout> 
            <p>{timeString}</p>
            <PostsContainer category=""/>
        </Layout>
    );
};

export async function getStaticProps({ params }) {
    //const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/posts/get?maxID=-1`);
    //const posts = await res.json();
    //console.log(posts)
    console.log("Rendering...")
    return {
      // Set the timeout for generating to 1 second
      // This timeout could be longer depending on how often data changes
      props: {
        
        time: Date.now()
      },
      unstable_revalidate: 10
    };
  }

export default Index;
