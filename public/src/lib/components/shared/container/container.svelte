<script lang="ts">
	let { children } = $props();
</script>

<div class="flex justify-center">
	<div class="w-11/12 max-w-7xl">
		{@render children()}
	</div>
</div>
