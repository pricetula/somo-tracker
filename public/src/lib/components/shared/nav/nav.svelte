<script>
	import Button from '$lib/components/ui/button/button.svelte';
	import Container from '../container/container.svelte';
	import Logo from '../logo/logo.svelte';
</script>

<nav class="py-2 fixed top-0 w-full z-10 bg-background">
	<Container class="flex justify-between items-center">
		<Logo />
		<Button size="sm">Join</Button>
	</Container>
</nav>
