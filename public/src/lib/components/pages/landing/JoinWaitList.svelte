<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';

	let email = $state('');
</script>

<form method="POST" action="?/subscribe" class="mb-2 flex max-w-95 flex-col gap-4 sm:flex-row">
	<Input type="email" name="email" bind:value={email} placeholder="Email" required />
	<Button type="submit">Join Waitlist</Button>
</form>
