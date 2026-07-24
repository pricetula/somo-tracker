<script lang="ts">
    import { Button } from "$lib/components/ui/button";
    import { Input } from "$lib/components/ui/input";

    let email = $state("");
</script>

<form
    method="POST"
    action="?/subscribe"
    class="flex flex-col sm:flex-row gap-4 max-w-95 mb-2"
>
    <Input
        type="email"
        name="email"
        bind:value={email}
        placeholder="Email"
        required
    />
    <Button type="submit">Join Waitlist</Button>
</form>
