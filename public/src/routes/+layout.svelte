<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Nav from '$lib/components/shared/nav/nav.svelte';
	import Footer from '$lib/components/shared/footer/footer.svelte';

	let { children } = $props();
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<main>
	<Nav />
	{@render children()}
	<Footer />
</main>
