<script lang="ts">
	let { children, class: className = '' } = $props();
</script>

<div class="flex justify-center">
	<div class="w-11/12 max-w-4xl {className}">
		{@render children()}
	</div>
</div>
