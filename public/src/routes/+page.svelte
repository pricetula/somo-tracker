<script lang="ts">
    import { Check } from "@lucide/svelte";
	import Container from '$lib/components/shared/container/container.svelte';
	import JoinWaitList from "$lib/components/pages/landing/JoinWaitList.svelte";
	import ValueShowcase from "$lib/components/pages/landing/ValueShowcase.svelte";
</script>

<article class="mt-30">
	<Container>
		<section id="hero" class="min-h-96 grid md:grid-cols-2 gap-8">
            <div class="order-2 lg:order-1">
                <h1 class="text-2xl font-bold tracking-tight leading-tight mb-4">
                    <span class="hero-line hero-line-2">Zero Waste.</span>
                    <br />
                    <span class="hero-line hero-line-1">Zero Printing.</span>
                    <br />
                    <span class="hero-line hero-line-3"> 100% Academic Insight.</span>
                </h1>

                <p class="mb-4 hero-p">
                    Eliminate administrative overhead and printing costs. Harness
                    real-time academic data insights to identify struggling students
                    early and guide them toward a successful, personalized learning
                    path.
                </p>

                <ul class="space-y-1 text-xs mb-8">
                    <li class="flex items-center gap-1 hero-check hero-check-1">
                        <Check size="14" class="text-green-500" />
                        <span>Assessments</span>
                    </li>
                    <li class="flex items-center gap-1 hero-check hero-check-2">
                        <Check size="14" class="text-green-500" />
                        <span>AI-Driven Gap Forecasting</span>
                    </li>
                    <li class="flex items-center gap-1 hero-check hero-check-3">
                        <Check size="14" class="text-green-500" />
                        <span>Setup in &lt; 2 hours</span>
                    </li>
                </ul>

                <div class="hero-cta">
                    <JoinWaitList />
                </div>
            </div>
            <div class="order-1 lg:order-2 bg-accent-foreground/5 h-70">
            </div>
        </section>

        <ValueShowcase />
	</Container>
</article>
 