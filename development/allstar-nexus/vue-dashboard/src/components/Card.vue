<template>
  <div class="card" :class="{ loading: isLoading }">
    <div class="card-header" v-if="title || $slots.header">
      <slot name="header">
        <h3>{{ title }}</h3>
      </slot>
      <div class="card-actions" v-if="$slots.actions">
        <slot name="actions"></slot>
      </div>
    </div>
    <div class="card-body">
      <slot></slot>
    </div>
    <div class="card-footer" v-if="$slots.footer">
      <slot name="footer"></slot>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: String,
  isLoading: { type: Boolean, default: false }
})
</script>

<style scoped>
.card {
  background: #1c1c1c;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  transition: all 0.3s ease;
}

.card:hover {
  border-color: #444;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.card.loading {
  opacity: 0.6;
  pointer-events: none;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid #2a2a2a;
}

.card-header h3 {
  margin: 0;
  font-size: 1.25rem;
  color: #eee;
  font-weight: 600;
}

.card-actions {
  display: flex;
  gap: 0.5rem;
}

.card-body {
  color: #ddd;
}

.card-footer {
  margin-top: 1rem;
  padding-top: 0.75rem;
  border-top: 1px solid #2a2a2a;
  font-size: 0.875rem;
  color: #999;
}
</style>
